# gnark ZK-SNARK Token Driver Design Document

## 1. Executive Summary

This document specifies the design and implementation of a gnark-based ZK-SNARK token driver
for the Hyperledger Fabric Token SDK. The driver introduces a new privacy-preserving token
driver using Groth16 proofs on BN254, replacing the existing zkatdlog driver's Groth-Sahai
and Schnorr proof system with a SNARK-native architecture that delivers reduction
in proof generation time and constant-time verification suitable for Fabric chaincode.

### Key Design Decisions

- **Proof System**: Groth16 on BN254 via gnark, replacing Groth-Sahai/Schnorr proofs over BLS12-381
- **Commitment Scheme**: Poseidon hash over the BN254 scalar field, replacing Pedersen commitments over BLS12-381; avoids emulated arithmetic inside R1CS circuits
- **Circuit Architecture**: ZCash Sapling multi-proof model: one independent circuit per input token (SpendCircuit) and one per output token (OutputCircuit); not a monolithic NIZK
- **Value Commitment**: ZCash Sapling `ValueCommit` construction over Jubjub (a twisted Edwards curve defined over the BN254 scalar field), enabling native curve operations inside BN254 circuits
- **Conservation Proof**: Homomorphic binding signature outside the circuits, identical to ZCash Sapling; endorsers sign `bsk = Σrcv_in − Σrcv_out`, proving `Σv_in = Σv_out` without revealing individual values
- **Backward Compatibility**: MigrationCircuit bridges existing zkatdlog Pedersen-committed tokens to the new Poseidon commitment scheme, enabling existing token holders to participate without reissuance
- **Trusted Setup**: Per-circuit Groth16 ceremony for Phase 1 (local for development); PLONK/KZG universal SRS evaluated as production alternative to eliminate per-circuit ceremonies
- **Phase 1 Scope**: Value privacy only, token amounts hidden, transaction graph visible, ownership enforced via Fabric MSP identity at the ledger layer
- **Phase 2 Scope**: Full privacy, transaction graph hidden via incremental Merkle tree accumulator and position-dependent nullifiers (ZCash Sapling `MixingPedersenHash`); ownership bound inside the commitment and proved in-circuit
- **Key Consistency Invariant**: Native Go implementations and gnark circuit gadgets for all primitives (Poseidon, Jubjub scalar multiplication, MixingPedersenHash) must use identical constants, encodings, and output extraction; enforced via shared test vectors

### Performance Targets

| Metric | Paper Baseline (Androulaki 2020) | Phase 1 Target | Phase 2 Target |
|--------|----------------------------------|----------------|----------------|
| Transfer proof generation (2-in/2-out) | ~1,993 ms | < 50 ms | < 100 ms |
| Proof verification (per circuit) | ~750 ms | < 3 ms | < 3 ms |
| Total validation (2-in/2-out) | ~3,000 ms | < 10 ms | < 15 ms |
| Proof size (per circuit) | — | 192 bytes | 192 bytes |
| Verification constant-time | No | Yes (Groth16) | Yes (Groth16) |

### Out of Scope for Phase 1

- Graph hiding (transaction graph is visible; deferred to Phase 2)
- In-circuit ownership proofs (enforced via Fabric MSP identity at ledger layer in Phase 1)
- Auditability circuits (deferred; the paper's auditability proof is the largest single cost)
- Batch proving or proof aggregation
- Proof compression

### Out of Scope for Phase 2

- Cross-driver interoperability beyond the MigrationCircuit
- zkEVM compatibility
- Threshold ownership (single owner per token)

### Design Review Status

**Last Review**: 2026-06-22
**Status**: Initial design document; circuit specifications and Phase 1 architecture confirmed
with mentor Angelo De Caro
**Reference**: ZCash Sapling specification (adopted for multi-proof model, value commitment
construction, and nullifier lifecycle)
**Benchmark Reference**: Androulaki et al., AFT 2020, Table 1 component breakdown used as
the performance baseline throughout

---

## 2. Architecture Overview

### 2.0 Driver Position in the SDK

The gnark driver is a token driver, analogous in role to the existing `zkatdlog` driver.
It sits at `token/core/zkat-snark-driver/` and implements the `driver.TokenDriver` interface.
It is independent of the network driver layer - the network driver (Fabric, FabricX,
Ethereum) handles transaction submission and ledger state, while the token driver handles
cryptographic validity of token operations.

```
Application
    │
    ▼
Token Management Service (TMS)
    │
    ├── token/core/zkat-snark-driver/   # this driver (new)
    │       ├── Prover (client-side)
    │       └── Validator (chaincode-side)
    │
    ├── token/core/zkatdlog/   # existing privacy driver
    └── token/core/fabtoken/   # existing cleartext driver
    │
    ▼
Network Driver (Fabric / FabricX / Ethereum)
```

### 2.1 High-Level Flow — Token Transfer

```
sequenceDiagram
    participant App as Application
    participant TMS as Token Management Service
    participant Prover as gnark Prover (client)
    participant Fabric as Fabric Network
    participant Validator as gnark Validator (chaincode)
    participant Ledger as Fabric Ledger

    Note over App,Ledger: Phase 1 Token Transfer (2-in, 2-out)

    App->>TMS: Transfer(inputs, recipients, amounts)
    TMS->>Prover: BuildTransferTransaction(inputs, outputs)

    Prover->>Prover: WitnessBuilder: build SpendCircuit{} x2, OutputCircuit{} x2

    par Parallel Proof Generation
        Prover->>Prover: SpendProver: groth16.Prove(SpendCircuit_0, pk_spend)
        Prover->>Prover: SpendProver: groth16.Prove(SpendCircuit_1, pk_spend)
        Prover->>Prover: OutputProver: groth16.Prove(OutputCircuit_0, pk_output)
        Prover->>Prover: OutputProver: groth16.Prove(OutputCircuit_1, pk_output)
    end

    Prover->>Prover: BindingSigComputer: bsk = Σrcv_in - Σrcv_out
    Prover->>Prover: BindingSigComputer: sig = JubjubSchnorr.Sign(bsk, txHash)
    Prover->>Prover: Orchestrator: assemble GnarkTransferTransaction{}
    Prover-->>TMS: GnarkTransferTransaction

    TMS->>Fabric: Submit transaction

    Note over Fabric,Ledger: Validation on endorsing peers

    Fabric->>Validator: ValidateTokenActions(tx)
    Validator->>Validator: Deserialise + structural checks
    Validator->>Validator: Build public witnesses x4
    par Parallel Verification
        Validator->>Validator: groth16.Verify(SpendProof_0, vkSpend, pw_0)
        Validator->>Validator: groth16.Verify(SpendProof_1, vkSpend, pw_1)
        Validator->>Validator: groth16.Verify(OutputProof_0, vkOutput, pw_0)
        Validator->>Validator: groth16.Verify(OutputProof_1, vkOutput, pw_1)
    end
    Validator->>Validator: bvk = Σcv_in - Σcv_out
    Validator->>Validator: JubjubSchnorr.Verify(sig, txHash, bvk)
    Validator->>Ledger: UTXO checks (existence, double-spend, type, ownership)
    Validator->>Ledger: Apply state transition
    Validator-->>Fabric: Valid

    Fabric-->>App: Transaction committed
```

### 2.2 Component Architecture

```
token/core/zkat-snark-driver/
├── driver/
│   └── driver.go              # Driver registration and factory methods
├── crypto/
│   ├── poseidon/
│   │   └── hash.go            # Native Go Poseidon (BN254 scalar field)
│   ├── jubjub/
│   │   ├── commitment.go      # ValueCommit: cv = v·V + rcv·R
│   │   └── schnorr.go         # Jubjub Schnorr sign/verify
│   └── params/
│       └── params.go          # PublicParams: V, R generator points + vk/pk management
├── circuit/
│   ├── spend.go               # SpendCircuit R1CS definition
│   ├── output.go              # OutputCircuit R1CS definition
│   ├── migration.go           # MigrationCircuit R1CS definition
│   └── gadgets/
│       ├── poseidon.go        # Poseidon gnark circuit gadget
│       ├── valuecommit.go     # Value commitment gnark gadget
│       └── nullifier.go       # MixingPedersenHash gnark gadget (Phase 2)
├── prover/
│   ├── witness.go             # WitnessBuilder: note → circuit assignment
│   ├── spend.go               # SpendProver: groth16.Prove for SpendCircuit
│   ├── output.go              # OutputProver: groth16.Prove for OutputCircuit
│   └── orchestrator.go        # Parallel orchestration, binding sig, tx assembly
├── verifier/
│   └── verifier.go            # Chaincode validator: groth16.Verify + UTXO checks
├── token/
│   ├── token.go               # Note struct, commitment scheme, serialization
│   ├── issue.go               # IssueAction implementation
│   ├── transfer.go            # TransferAction implementation
│   └── wallet.go              # WalletService: UTXO selection, key management
├── pp/
│   └── pp.go                  # PublicParameters: setup, serialization, distribution
├── setup/
│   └── setup.go               # Trusted setup utilities (groth16.Setup wrappers)
└── graph/                     # Phase 2: Graph hiding extension
    ├── tree/
    │   └── tree.go            # IncrementalMerkleTree accumulator
    ├── nullifier/
    │   └── nullifier.go       # NullifierHash: MixingPedersenHash(cm, pos)
    └── circuit/
        ├── membership.go      # Merkle path verification gadget
        └── extended_spend.go  # ExtendedSpendCircuit (Phase 2)
```

---

## 3. Cryptographic Primitives

### 3.1 Token Commitment: Poseidon Hash

The token commitment replaces the Pedersen commitment used by the zkatdlog driver.
Poseidon is a SNARK-friendly sponge hash over native BN254 scalar field elements.
It requires ~240–300 constraints per evaluation inside a BN254 Groth16 circuit, compared
to thousands for SHA-256 or Pedersen over a non-native curve.

**Phase 1** (3-input variant, capacity t = 4):
```
cm = Poseidon(v, type, r)
```

**Phase 2** (5-input variant, capacity t = 6, owner bound inside commitment):
```
cm = Poseidon(v, type, r, owner_pk_u, owner_pk_v)
```

**Merkle node hash** (2-input variant with domain separation, Phase 2):
```
node = Poseidon(domain, left, right)
  where domain = 0 for internal nodes, 1 for leaf nodes
```

**Implementation**:
```go
// token/core/zkat-snark-driver/crypto/poseidon/hash.go

// Hash computes Poseidon over BN254 scalar field elements.
// inputs must match the capacity of the chosen variant (3 for Phase 1, 5 for Phase 2).
// Round constants are generated from the canonical seed defined in params.go.
func Hash(inputs ...fr.Element) (fr.Element, error) {
    if len(inputs) == 0 || len(inputs) > MaxInputs {
        return fr.Element{}, ErrInvalidInputCount
    }
    state := initState(len(inputs))
    copy(state[1:], inputs)
    return permute(state), nil
}

// HashCircuit is the gnark gadget equivalent of Hash.
// It must use identical round constants and output extraction as Hash.
// Cross-tested via shared test vectors in poseidon_test.go.
func HashCircuit(api frontend.API, inputs ...frontend.Variable) (frontend.Variable, error) {
    // ... gnark gadget implementation
}
```

**Key consistency invariant**: `Hash` and `HashCircuit` must produce identical outputs
for identical inputs. This is the single most critical correctness requirement in the
entire driver, any discrepancy breaks every proof for every transaction. Enforced by
a dedicated cross-implementation test suite (`TestPoseidonCrossConsistency`) that runs
the same inputs through both implementations and compares outputs.

### 3.2 Value Commitment: ZCash Sapling ValueCommit

The value commitment binds a token's value to a homomorphic structure that enables
conservation proofs without revealing individual values. The Jubjub curve is a twisted
Edwards curve defined over the BN254 scalar field, making Jubjub operations native
(non-emulated) inside BN254 Groth16 circuits.

```
cv = v · V + rcv · R   (over Jubjub)
```

Where `V` and `R` are public generator points derived from a nothing-up-my-sleeve
procedure: no entity knows their discrete logarithm relationship to the Jubjub base point.

**Value commitment properties**:
- **Homomorphic**: `cv_1 + cv_2 = (v_1 + v_2)·V + (rcv_1 + rcv_2)·R`
- **Binding**: computational difficulty of producing two openings for the same commitment (DL assumption on Jubjub)
- **Hiding**: `rcv` is uniform random, so `cv` reveals nothing about `v`

**Binding signature for conservation**:

```
bsk = Σrcv_in − Σrcv_out         (private, computed by prover)
bvk = Σcv_in  − Σcv_out          (public, computable from transaction data)
    = (Σv_in − Σv_out)·V + bsk·R

sig = JubjubSchnorr.Sign(bsk, txHash)
```

The signature verifies against `bvk` if and only if `(Σv_in − Σv_out)·V = 0`.
Since `V` generates a prime-order Jubjub subgroup, this holds iff `Σv_in = Σv_out`.

**Implementation**:
```go
// token/core/zkat-snark-driver/crypto/jubjub/commitment.go

type PublicParams struct {
    V twistededwards.PointAffine  // Value generator
    R twistededwards.PointAffine  // Randomness generator
}

// ValueCommit computes cv = v·V + rcv·R over Jubjub.
// v is a uint64 token denomination; rcv is 32-byte uniform random scalar.
func ValueCommit(v uint64, rcv []byte, pp *PublicParams) (twistededwards.PointAffine, error) {
    var vScalar, rcvScalar fr.Element
    vScalar.SetUint64(v)
    if err := rcvScalar.SetBytesCanonical(rcv); err != nil {
        return twistededwards.PointAffine{}, ErrInvalidScalar
    }
    curve := twistededwards.GetCurve(twistededwards.BabyJubJub)
    termV := curve.ScalarMul(&pp.V, &vScalar)
    termR := curve.ScalarMul(&pp.R, &rcvScalar)
    var cv twistededwards.PointAffine
    cv.Add(&termV, &termR)
    return cv, nil
}
```

### 3.3 Public Parameters

The public parameters are distributed via the Fabric channel configuration and loaded
by both the prover (client) and the validator (chaincode).

```go
// token/core/zkat-snark-driver/pp/pp.go

type PublicParams struct {
    // Curve and field
    Curve       ecc.ID                       // BN254

    // Value commitment generators (Jubjub)
    V           twistededwards.PointAffine   // value generator
    R           twistededwards.PointAffine   // randomness generator

    // Nullifier generators (Phase 2 only)
    // 106 Jubjub points for MixingPedersenHash windowed computation
    NullifierGenerators []twistededwards.PointAffine

    // Proving keys (one per circuit; distributed to clients)
    PKSpend     []byte  // groth16 proving key for SpendCircuit
    PKOutput    []byte  // groth16 proving key for OutputCircuit
    PKMigration []byte  // groth16 proving key for MigrationCircuit

    // Verification keys (one per circuit; stored in channel config)
    VKSpend     []byte  // groth16 verification key for SpendCircuit
    VKOutput    []byte  // groth16 verification key for OutputCircuit
    VKMigration []byte  // groth16 verification key for MigrationCircuit

    // Merkle tree parameters (Phase 2 only)
    TreeDepth   int     // 32 (supports 2^32 ≈ 4B token slots)
}
```

**Setup and distribution**:
- Proving keys and verification keys are generated once per circuit via `groth16.Setup`
- The verification keys are small (~1 KB each) and stored in the Fabric channel configuration
- The proving keys are larger (~400–600 KB each) and distributed to clients out-of-band or via the TMS
- For Phase 1, a local trusted setup ceremony is used for development and benchmarking
- For production, options are: (a) multi-party Groth16 ceremony per circuit, or (b) PLONK/KZG with a universal SRS to eliminate per-circuit ceremonies - deferred decision

```go
// Serialize/deserialize for channel config distribution
func (pp *PublicParams) Serialize() ([]byte, error)
func DeserializePublicParams(raw []byte) (*PublicParams, error)

// Compute a deterministic hash of the public params for endorser consistency checks
func (pp *PublicParams) Hash() ([32]byte, error) {
    raw, err := pp.Serialize()
    if err != nil {
        return [32]byte{}, err
    }
    return sha256.Sum256(raw), nil
}
```

---

## 4. Circuit Layer

### 4.1 Circuit Architecture: Multi-Proof Model

The gnark driver uses the ZCash Sapling multi-proof model: one independent circuit per
input token and one per output token. This is a deliberate rejection of the paper's
monolithic NIZK for four reasons:

- **Parallelism**: each circuit proof is independent and generated concurrently
- **Constant verification cost**: Groth16 verification is always four pairings per proof, regardless of circuit size
- **Modular extensibility**: Phase 2 adds constraints only to SpendCircuit; OutputCircuit changes minimally; MigrationCircuit is unaffected
- **Independent trusted setups**: circuits can be updated and re-setup independently

### 4.2 SpendCircuit

The SpendCircuit proves that the prover knows a valid private opening of a token
commitment already recorded on the ledger. It enforces two constraint groups:

*Public inputs*: visible to the verifier and recorded in the proof:
- CommitmentIn — the Poseidon hash of the input token
- ValueCommitInX, ValueCommitInY — Jubjub coordinates of the value commitment

*Private inputs*: known only to the prover, never published:
- Value — token denomination
- TokenType — asset type
- Randomness — commitment randomness r
- RCV — value commitment randomness

*Constraints*:
1. CommitmentIn == Poseidon(Value, TokenType, Randomness) - proves the prover knows a valid opening of the on-ledger commitment
2. (ValueCommitInX, ValueCommitInY) == Value·V + RCV·R - proves the published value commitment correctly encodes the same value

```go
// token/core/zkat-snark-driver/circuit/spend.go

type SpendCircuit struct {
    // ── Public inputs ──────────────────────────────────────
    CommitmentIn   frontend.Variable `gnark:",public"`  // cm = Poseidon(v, type, r); 32 bytes
    ValueCommitInX frontend.Variable `gnark:",public"`  // cv.X: Jubjub U-coordinate of v·V + rcv·R
    ValueCommitInY frontend.Variable `gnark:",public"`  // cv.Y: Jubjub V-coordinate

    // ── Private inputs ─────────────────────────────────────
    Value      frontend.Variable  // token denomination v
    TokenType  frontend.Variable  // asset type encoded as field element
    Randomness frontend.Variable  // commitment randomness r
    RCV        frontend.Variable  // value commitment randomness rcv
}
```

**Constraint budget**:
| Group | Operation | Constraints |
|-------|-----------|-------------|
| 1 | Poseidon(v, type, r), t=4 | ~250–280 |
| 2 | Two Jubjub scalar muls + point add | ~1,800 |
| **Total** | | **~2,050–2,080** |

> Note: Earlier estimates of ~350 total were for Poseidon alone. The value commitment
> group (two Jubjub scalar multiplications) dominates at ~1,800 constraints. Total
> circuit cost is ~2,050 constraints. Prove time < 5 ms.

### 4.3 OutputCircuit

The OutputCircuit proves that a newly created output token is well-formed and carries
a positive denomination. It is structurally identical to the SpendCircuit with one
additional constraint group:

*Public inputs*: CommitmentOut, ValueCommitOutX, ValueCommitOutY

*Private inputs*: Value, TokenType, Randomness, RCV

*Constraints*:
1. CommitmentOut == Poseidon(Value, TokenType, Randomness)
2. (ValueCommitOutX, ValueCommitOutY) == Value·V + RCV·R
3. Value ∈ [1, 2^64) - non-negativity and non-zero check; prevents zero-value and
   artificially large token creation


```go
// token/core/zkat-snark-driver/circuit/output.go

type OutputCircuit struct {
    // ── Public inputs ──────────────────────────────────────
    CommitmentOut   frontend.Variable `gnark:",public"`
    ValueCommitOutX frontend.Variable `gnark:",public"`
    ValueCommitOutY frontend.Variable `gnark:",public"`

    // ── Private inputs ─────────────────────────────────────
    Value      frontend.Variable
    TokenType  frontend.Variable
    Randomness frontend.Variable
    RCV        frontend.Variable
}
```

**Constraint budget**:
| Group | Operation | Constraints |
|-------|-----------|-------------|
| 1 | Poseidon(v, type, r), t=4 | ~250–280 |
| 2 | Two Jubjub scalar muls + point add | ~1,800 |
| 3 | Range check (64-bit decomposition) | ~130 |
| **Total** | | **~2,180–2,210** |

### 4.4 MigrationCircuit

The MigrationCircuit enables holders of existing zkatdlog Pedersen-committed tokens to
spend those tokens under the new gnark driver. It bridges the two commitment schemes by
proving that a Pedersen commitment and a Poseidon commitment open to the same (v, type).
The owner never needs to reissue their tokens.

*Public inputs*: CommitmentPedersenX, CommitmentPedersenY (existing on-ledger commitment), CommitmentPoseidon (new commitment to be registered)

*Private inputs*: Value, TokenType, RandomnessPed (Pedersen randomness), RandomnessHash (Poseidon randomness)

*Constraints*:
1. (CommitmentPedersenX, CommitmentPedersenY) == Value·G + RandomnessPed·H — proves the prover knows the opening of the existing zkatdlog commitment
2. CommitmentPoseidon == Poseidon(Value, TokenType, RandomnessHash) — proves the new Poseidon commitment is correctly formed
3. Value and TokenType are shared variables across both groups — this is the binding constraint that makes it impossible to bridge a different token than the one being migrated

Pedersen over BLS12-381 requires emulated arithmetic inside the BN254 circuit, making
this the most constraint-heavy circuit (~2,480 total).

```go
// token/core/zkat-snark-driver/circuit/migration.go

type MigrationCircuit struct {
    // ── Public inputs ──────────────────────────────────────
    // The existing zkatdlog Pedersen commitment: cm_ped = v·G + r_ped·H
    CommitmentPedersenX frontend.Variable `gnark:",public"`
    CommitmentPedersenY frontend.Variable `gnark:",public"`
    // The new gnark Poseidon commitment: cm_hash = Poseidon(v, type, r_hash)
    CommitmentPoseidon  frontend.Variable `gnark:",public"`

    // ── Private inputs ─────────────────────────────────────
    Value          frontend.Variable  // token denomination v
    TokenType      frontend.Variable  // asset type
    RandomnessPed  frontend.Variable  // Pedersen randomness r_ped
    RandomnessHash frontend.Variable  // Poseidon randomness r_hash
}
```

**Constraint budget**:
| Group | Operation | Constraints |
|-------|-----------|-------------|
| 1 | Pedersen commitment (emulated BLS12-381 ops) | ~2,000–2,200 |
| 2 | Poseidon(v, type, r_hash), t=4 | ~250–280 |
| 3 | Shared variable binding (no extra constraints) | 0 |
| **Total** | | **~2,250–2,480** |

Prove time < 30 ms; verify time ~2 ms.

### 4.5 Circuit Specifications Summary

| | SpendCircuit | OutputCircuit | MigrationCircuit |
|--|--|--|--|
| **Purpose** | Prove valid spend of existing token | Prove well-formed new output token | Bridge Pedersen → Poseidon commitment |
| **Public inputs** | CommitmentIn, ValueCommitInX/Y | CommitmentOut, ValueCommitOutX/Y | CommitmentPedersenX/Y, CommitmentPoseidon |
| **Private inputs** | v, type, r, rcv | v, type, r, rcv | v, type, r_ped, r_hash |
| **Constraint count** | ~2,080 | ~2,210 | ~2,480 |
| **Prove time** | < 5 ms | < 5 ms | < 30 ms |
| **Verify time** | ~2 ms | ~2 ms | ~2 ms |

### 4.6 Circuit Correctness Requirements

Every circuit must satisfy the following test categories before being used in integration:

| Category | Requirement | Pass Criterion |
|----------|-------------|----------------|
| Valid witness | Correct private inputs produce a proof | `groth16.Verify` returns nil |
| Wrong value | Incorrect v that does not match CommitmentIn | `groth16.Prove` returns unsatisfied constraint |
| Wrong randomness | Incorrect r that does not open CommitmentIn | `groth16.Prove` returns unsatisfied constraint |
| Wrong value commitment | cv that does not correspond to v and rcv | `groth16.Prove` returns unsatisfied constraint |
| Zero-value output | v = 0 in OutputCircuit | `groth16.Prove` returns unsatisfied constraint |
| Constraint count log | Logged via `ccs.GetNbConstraints()` | Count matches documented budget ±5% |

Invalid-witness tests are mandatory. A circuit that accepts invalid witnesses is worse
than no circuit: it means proofs cannot be trusted even when they verify.

---

## 5. Token Representation

### 5.1 Note Structure

Following ZCash Sapling terminology, a "note" is the complete private description of a
token, i.e., all information needed to spend it.

**Phase 1 Note**:
```go
// token/core/zkat-snark-driver/token/token.go

type Note struct {
    Value      uint64  // token denomination (hidden)
    TokenType  string  // asset type identifier (hidden)
    Randomness []byte  // 32-byte uniform random commitment randomness (secret)
    // Owner is not inside the commitment in Phase 1.
    // Ownership is enforced via Fabric MSP identity at the ledger layer.
}

// Commitment computes cm = Poseidon(v, type, r)
func (n *Note) Commitment() (fr.Element, error) {
    var v, t, r fr.Element
    v.SetUint64(n.Value)
    t.SetBytes([]byte(n.TokenType))
    if err := r.SetBytesCanonical(n.Randomness); err != nil {
        return fr.Element{}, ErrInvalidRandomness
    }
    return poseidon.Hash(v, t, r)
}
```

**Phase 2 Note** (owner bound inside commitment):
```go
type Note struct {
    Value      uint64
    TokenType  string
    Randomness []byte
    OwnerPk    twistededwards.PointAffine  // Jubjub public key of owner
    // OwnerSk is NOT stored in the Note; it lives in the wallet's key material
}

// Commitment computes cm = Poseidon(v, type, r, owner_pk_u, owner_pk_v)
func (n *Note) Commitment() (fr.Element, error) {
    var v, t, r fr.Element
    v.SetUint64(n.Value)
    t.SetBytes([]byte(n.TokenType))
    if err := r.SetBytesCanonical(n.Randomness); err != nil {
        return fr.Element{}, ErrInvalidRandomness
    }
    return poseidon.Hash(v, t, r, n.OwnerPk.X, n.OwnerPk.Y)
}
```

### 5.2 Encrypted Note Transmission

When a sender creates an output token for a recipient, the note secrets must be
transmitted privately. The sender encrypts the note using ephemeral ECDH over Jubjub
against the recipient's Jubjub public key, then includes the ciphertext in the transaction.

```go
type EncryptedNote struct {
    EphemeralPkX []byte  // 32 bytes: ephemeral Jubjub public key X
    EphemeralPkY []byte  // 32 bytes: ephemeral Jubjub public key Y
    Ciphertext   []byte  // AES-256-GCM encrypted Note
    // Phase 2 also carries the leaf position in the Merkle tree
}
```

### 5.3 Token ID Mapping

The SDK uses composite string keys internally: `\x00token\x00<txID>\x00<index>`.
The gnark driver maps these to 32-byte field elements for on-ledger storage:

```go
// ComputeOutputID maps an SDK token ID to the on-ledger bytes32 key.
// Deterministic and collision-resistant.
func ComputeOutputID(txID string, index uint64) ([32]byte, error) {
    var txIDField, indexField fr.Element
    txIDField.SetBytes([]byte(txID))
    indexField.SetUint64(index)
    cm, err := poseidon.Hash(txIDField, indexField)
    if err != nil {
        return [32]byte{}, err
    }
    return cm.Bytes(), nil
}
```

---

## 6. Proof Generation

### 6.1 WitnessBuilder

The WitnessBuilder is the boundary between the application layer (which works with Notes
and token amounts) and the circuit layer (which works with field elements and gnark
Variables). It translates a Note and its associated randomness values into a typed
gnark circuit assignment struct, encoding each field as a BN254 scalar field element using
canonical big-endian byte encoding.

The encoding used here and the encoding used by the validator's public witness reconstructor
must be byte-for-byte identical. This is the most common source of integration failure
when connecting the prover and verifier. Both use fr.Element.SetBytesCanonical(), which
additionally rejects byte sequences representing integers ≥ the field modulus.

```go
// token/core/zkat-snark-driver/prover/witness.go

func BuildSpendWitness(
    note *token.Note,
    rcv []byte,
    pp *params.PublicParams,
) (*circuit.SpendCircuit, error) {
    // 1. Derive commitment (cross-checked against ledger)
    cm, err := note.Commitment()
    if err != nil {
        return nil, fmt.Errorf("commitment derivation: %w", err)
    }

    // 2. Compute value commitment cv = v·V + rcv·R
    cv, err := jubjub.ValueCommit(note.Value, rcv, &pp.ValueCommitParams)
    if err != nil {
        return nil, fmt.Errorf("value commitment: %w", err)
    }

    // 3. Encode field elements - MUST use identical encoding as public witness builder in verifier
    var vField, typeField, rField, rcvField fr.Element
    vField.SetUint64(note.Value)
    typeField.SetBytes([]byte(note.TokenType))
    if err := rField.SetBytesCanonical(note.Randomness); err != nil {
        return nil, fmt.Errorf("randomness encoding: %w", err)
    }
    if err := rcvField.SetBytesCanonical(rcv); err != nil {
        return nil, fmt.Errorf("rcv encoding: %w", err)
    }

    return &circuit.SpendCircuit{
        CommitmentIn:   cm,
        ValueCommitInX: cv.X,
        ValueCommitInY: cv.Y,
        Value:          vField,
        TokenType:      typeField,
        Randomness:     rField,
        RCV:            rcvField,
    }, nil
}
```

**Encoding invariant**: `rField.SetBytesCanonical(note.Randomness)` in the WitnessBuilder
and `fr.Element.SetBytesCanonical(input.CommitmentRandomness)` in the verifier's public
witness builder must use identical encoding. Any discrepancy - endianness, canonical vs.
non-canonical, field vs. byte slice - breaks every proof silently.

### 6.2 SpendProver and OutputProver

Each prover holds a proving key loaded once at startup and reused across transactions.
The proving key is read-only, so multiple goroutines can call groth16.Prove concurrently
without synchronization, each invocation constructs its own witness object and
intermediate state locally.

```go
// token/core/zkat-snark-driver/prover/spend.go

type SpendProver struct {
    pk groth16.ProvingKey  // loaded once at startup, reused across transactions
}

func (p *SpendProver) Prove(assignment *circuit.SpendCircuit) (groth16.Proof, error) {
    witness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField())
    if err != nil {
        return nil, fmt.Errorf("witness construction: %w", err)
    }
    proof, err := groth16.Prove(p.cs, p.pk, witness)
    if err != nil {
        return nil, fmt.Errorf("groth16.Prove: %w", err)
    }
    return proof, nil
}
```

The proving key is read-only after loading. Multiple goroutines can call `Prove`
concurrently without synchronization, each call constructs its own witness object
and intermediate state.

### 6.3 Parallel Orchestrator

The Orchestrator drives parallel proof generation, collects outputs, and assembles the
final transaction. Steps 2a and 2b (spend and output proving) are fully independent and
run concurrently in separate goroutines via an errgroup. The total client-side prove time
is therefore max(SpendProve, OutputProve) rather than their sum.

*Orchestration flow*:
1. Call WitnessBuilder for each input and output to produce circuit assignments
2. Launch one goroutine per circuit: groth16.Prove(assignment, pk) for each SpendCircuit and OutputCircuit concurrently
3. Wait for all goroutines; fail fast on any error
4. Collect all rcv values from completed proofs; compute bsk = Σrcv_in − Σrcv_out
5. Compute txHash (canonical hash of all public commitments and value commitments, sorted deterministically)
6. Compute binding signature: JubjubSchnorr.Sign(bsk, txHash)
7. Assemble and return GnarkTransferTransaction

### 6.4 Binding Signature

The binding signature is computed from the randomness values accumulated across all
input and output proofs. It does not require any additional circuit work.


bsk = Σrcv_in − Σrcv_out
txHash = SHA256("gnark-token-transfer-v1\x00" || sorted_inputs || sorted_outputs)
sig = JubjubSchnorr.Sign(bsk, txHash)


The txHash must be computed identically by the prover and the validator. Inputs and
outputs are sorted lexicographically by commitment before hashing to ensure the hash is
canonical regardless of goroutine completion order. EncryptedNote is intentionally
excluded from the hash as it is recipient metadata and does not affect cryptographic validity.

```go
func (o *Orchestrator) computeBindingSignature(
    spends  []proofResult,
    outputs []proofResult,
    pp      *params.PublicParams,
) ([]byte, error) {
    // bsk = Σrcv_in − Σrcv_out
    var bsk fr.Element
    for _, r := range spends {
        var rcvField fr.Element
        rcvField.SetBytesCanonical(r.rcv)
        bsk.Add(&bsk, &rcvField)
    }
    for _, r := range outputs {
        var rcvField fr.Element
        rcvField.SetBytesCanonical(r.rcv)
        bsk.Sub(&bsk, &rcvField)
    }

    txHash := computeTransactionHash(spends, outputs)
    return jubjub.SchnorrSign(bsk, txHash, pp)
}
```

### 6.5 Transaction Wire Format

The assembled transaction contains only public data, no private witness fields ever
leave the client.

```go
type GnarkTransferTransaction struct {
    TokenType        string
    SenderSignature  []byte  // Fabric MSP signature

    Inputs  []SpendDescription
    Outputs []OutputDescription

    BindingSignature []byte  // 64 bytes: Jubjub Schnorr (R_sig, s_sig)
}

type SpendDescription struct {
    CommitmentIn   []byte  // 32 bytes: Poseidon(v, type, r)
    ValueCommitInX []byte  // 32 bytes: Jubjub U-coordinate of cv_in
    ValueCommitInY []byte  // 32 bytes: Jubjub V-coordinate of cv_in
    SpendProof     []byte  // 192 bytes: Groth16 proof (A ∈ G1, B ∈ G2, C ∈ G1)
}

type OutputDescription struct {
    CommitmentOut   []byte  // 32 bytes
    ValueCommitOutX []byte  // 32 bytes
    ValueCommitOutY []byte  // 32 bytes
    OutputProof     []byte  // 192 bytes
    EncryptedNote   []byte  // ~96 bytes: AES-256-GCM encrypted note for recipient
    Recipient       []byte  // Fabric MSP identity of intended recipient
}
```

**Transaction hash computation** (must be deterministic and identical in prover and verifier):
```go
func computeTransactionHash(spends []proofResult, outputs []proofResult) []byte {
    h := sha256.New()
    h.Write([]byte("gnark-token-transfer-v1\x00"))
    // Inputs sorted lexicographically by commitment to ensure canonical ordering
    // regardless of the order in which goroutines returned
    for _, s := range sortedByCommitment(spends) {
        h.Write(s.cm.Marshal())
        h.Write(s.cv.X.Marshal())
        h.Write(s.cv.Y.Marshal())
    }
    for _, o := range sortedByCommitment(outputs) {
        h.Write(o.cm.Marshal())
        h.Write(o.cv.X.Marshal())
        h.Write(o.cv.Y.Marshal())
        // EncryptedNote is intentionally excluded: it is recipient metadata
        // and does not affect cryptographic validity of the transaction
    }
    return h.Sum(nil)
}
```

---

## 7. Chaincode Validator

### 7.1 Validation Architecture

The validator runs inside Fabric chaincode on every endorsing peer during the Validate
phase of the Execute-Order-Validate pipeline. It is the gatekeeper between a submitted
transaction and the ledger. It never touches proving keys or private witness data —
its only inputs are the serialised transaction and the verification keys loaded from the
channel configuration.

The critical ordering principle: cryptographic verification always runs before ledger reads.
An adversary submitting transactions with invalid proofs is rejected at the proof
verification step before a single state-DB read is attempted, preventing denial-of-service
on ledger I/O.

### 7.2 Full Validation Sequence

| Step | Action | Detail | Time |
|------|--------|--------|------|
| 1 | Deserialise | Parse transaction bytes; validate all field lengths: CommitmentIn 32 B, SpendProof 192 B, binding sig 64 B. Reject on any structural violation. | ~0.3 ms |
| 2 | Deserialise proofs | Decode each 192-byte proof into groth16.Proof: curve point decoding and on-curve check. Reject if any point is not on BN254. | ~0.5 ms |
| 3 | Build public witnesses | fr.Element.SetBytesCanonical() for each commitment and value commitment coordinate. Reject non-canonical encodings and off-curve Jubjub points. | ~0.2 ms |
| 4 | Verify all proofs (parallel) | One goroutine per proof: groth16.Verify(proof, vk, publicWitness). Reject if any returns an error. | ~2 ms |
| 5 | Compute bvk | bvk = Σcv_in − Σcv_out via Jubjub point arithmetic on public data only. No private data needed. | ~0.1 ms |
| 6 | Verify binding signature | Recompute txHash (canonical, identical construction to prover). JubjubSchnorr.Verify(sig, txHash, bvk). | ~0.5 ms |
| 7 | UTXO ledger checks | For each cm_in: exists in active set; not already spent; type matches; sender is owner. For each cm_out: not already in active or spent sets. | ~1 ms |
| 8 | Apply state transition | Mark all cm_in as spent; add all cm_out to active token set with recipient as owner. | ~0.5 ms |

### 7.3 The groth16.Verify Equation

The verification equation for Groth16 on BN254 is:


e(A, B) = e(α, β) · e(L, γ) · e(C, δ)

where L = γ_0 + l_1·γ_1 + l_2·γ_2 + l_3·γ_3
      l_i = public input field elements (CommitmentIn, ValueCommitInX, ValueCommitInY)
      γ_i = elements of gamma_abc array in the verification key
      A, B, C = the three proof points (from the 192-byte proof)


This requires exactly four pairing evaluations on BN254 (~2 ms total), regardless of
the circuit's constraint count. A 2,080-constraint SpendCircuit and a 15,130-constraint
Phase 2 SpendCircuit both verify in the same time. This is the succinctness property
that makes Groth16 compatible with Fabric's block commit latency budget.

The term L is where the public inputs enter the equation. It is different for every
transaction because the public inputs are different. A proof generated for CommitmentIn = 0x7f3a...
will only satisfy the equation when l_1 is set to that same value — changing the public
input in the witness changes L and breaks the equation. This is what makes public
inputs binding.

### 7.4 Public Witness Reconstruction

The validator rebuilds the public witness from the transaction's public fields.
The encoding must be byte-for-byte identical to the WitnessBuilder in Section 6.1.

```go
func (v *Verifier) buildSpendPublicWitness(desc *SpendDescription) (witness.Witness, error) {
    var cm, cvX, cvY fr.Element
    // SetBytesCanonical rejects non-canonical encodings (values >= field modulus)
    if err := cm.SetBytesCanonical(desc.CommitmentIn); err != nil {
        return nil, fmt.Errorf("CommitmentIn not a canonical field element: %w", err)
    }
    if err := cvX.SetBytesCanonical(desc.ValueCommitInX); err != nil {
        return nil, fmt.Errorf("ValueCommitInX not canonical: %w", err)
    }
    if err := cvY.SetBytesCanonical(desc.ValueCommitInY); err != nil {
        return nil, fmt.Errorf("ValueCommitInY not canonical: %w", err)
    }

    // Also verify the Jubjub point is on the curve before passing to verify
    pt := twistededwards.PointAffine{X: cvX, Y: cvY}
    if !twistededwards.BabyJubJub.IsOnCurve(&pt) {
        return nil, ErrValueCommitmentOffCurve
    }

    assignment := &circuit.SpendCircuit{
        CommitmentIn:   cm,
        ValueCommitInX: cvX,
        ValueCommitInY: cvY,
        // Private fields are zero — frontend.PublicOnly() extracts only public fields
    }
    return frontend.NewWitness(assignment, ecc.BN254.ScalarField(), frontend.PublicOnly())
}
```

### 7.5 UTXO Ledger Checks

```go
func (v *Verifier) validateLedgerState(tx *GnarkTransferTransaction) error {
    for i, input := range tx.Inputs {
        // Check 1: token exists in active set
        record, err := v.ledger.GetToken(input.CommitmentIn)
        if err != nil {
            return fmt.Errorf("input %d ledger read: %w", i, err)
        }
        if record == nil {
            return fmt.Errorf("input %d: commitment %x not found on ledger", i, input.CommitmentIn)
        }
        // Check 2: token is not already spent (double-spend prevention)
        spent, err := v.ledger.IsSpent(input.CommitmentIn)
        if err != nil {
            return fmt.Errorf("input %d spent check: %w", i, err)
        }
        if spent {
            return fmt.Errorf("input %d: commitment %x already spent", i, input.CommitmentIn)
        }
        // Check 3: token type matches transaction
        if record.TokenType != tx.TokenType {
            return fmt.Errorf("input %d: type mismatch (record=%s, tx=%s)", i, record.TokenType, tx.TokenType)
        }
        // Check 4: sender owns the token (Phase 1: Fabric MSP identity check)
        if !bytes.Equal(record.Owner, tx.SenderIdentity()) {
            return fmt.Errorf("input %d: sender does not own this token", i)
        }
    }

    for j, output := range tx.Outputs {
        // Check 5: output commitment does not already exist
        existing, err := v.ledger.GetToken(output.CommitmentOut)
        if err != nil {
            return fmt.Errorf("output %j ledger read: %w", j, err)
        }
        if existing != nil {
            return fmt.Errorf("output %j: commitment %x already exists", j, output.CommitmentOut)
        }
    }

    return nil
}
```

### 7.6 Correctness Invariants

| Invariant | What Breaks If Violated |
|-----------|------------------------|
| Verification key matches current circuit | Every legitimate transaction rejected after any circuit change |
| Public witness encoding matches prover encoding | Every legitimate proof fails, most common integration bug |
| `txHash` construction identical in prover and verifier | Every legitimate binding signature fails verification |
| Double-spend check (spent set lookup) | Same token spendable multiple times, token inflation |
| Output freshness check (cm_out not already on ledger) | Two tokens sharing a commitment, UTXO model corrupted |
| Token type homogeneity | Cross-asset value transfer permitted, semantic integrity broken |
| Phase 1 ownership: sender == `record.Owner` | Any party can spend any known commitment |
| Phase 2 ownership: in-circuit `owner_sk · G == owner_pk` in cm | Same as above, but circuit-enforced not ledger-enforced |

---

## 8. Driver Interface Implementation

### 8.1 TokenDriver Interface

The gnark driver implements the `driver.TokenDriver` interface in the same way as
the existing `zkatdlog` driver, registering under the driver name `"gnark"`.

```go
// token/core/zkat-snark-driver/driver/driver.go

const DriverName = "gnark"

type Driver struct{}

func (d *Driver) NewTokenService(
    tmsID            token.TMSID,
    publicParameters []byte,
    identityProvider driver.IdentityProvider,
    storageProvider  driver.StorageProvider,
    configuration    driver.Configuration,
) (driver.TokenManagerService, error)

func (d *Driver) NewValidator(
    tmsID            token.TMSID,
    publicParameters []byte,
) (driver.Validator, error)

func (d *Driver) NewPublicParametersManager(
    publicParameters []byte,
) (driver.PublicParametersManager, error)
```

Each method deserializes the public parameters from channel configuration and constructs
the appropriate component. The public parameters are the single shared artifact between
the prover (client) and validator (chaincode), if they differ, all proofs fail.

### 8.2 Issue Action

For issuance, no SpendCircuit proofs are required as there are no input tokens being
consumed. Only OutputCircuit proofs are generated (one per issued token). The binding
signature key for issuance is bsk = -Σrcv_out (no input RCVs), proving that the value
commitments in the output tokens are correctly formed without requiring conservation.
Issuance authorization comes from the Fabric MSP endorsement policy, not from a ZK proof.

```go
// token/core/zkat-snark-driver/token/issue.go

type IssueAction struct {
    Issuer   []byte
    Outputs  []*IssueOutput
}

type IssueOutput struct {
    Recipient   []byte  // Fabric MSP identity of recipient
    Note        *Note   // private note data (stays with issuer and recipient)
    Commitment  []byte  // 32-byte Poseidon commitment (public)
}

func (a *IssueAction) Validate() error {}
```

### 8.3 Transfer Action

A transfer action holds the private SpendInput records for each consumed token and
the TransferOutput records for each new token. The transfer action calls the Orchestrator
to build all proofs in parallel, compute the binding signature, and assemble the final
GnarkTransferTransaction.

```go
// token/core/zkat-snark-driver/token/transfer.go

type TransferAction struct {
    Inputs  []*SpendInput   // existing tokens being consumed
    Outputs []*TransferOutput
}

type SpendInput struct {
    TokenID    *token.ID              // SDK token ID (for UTXO lookup)
    Note       *Note                  // private note (known to spender)
    RCV        []byte                 // value commitment randomness for this input
}

type TransferOutput struct {
    Recipient  []byte
    Note       *Note
    RCV        []byte
}

func (a *TransferAction) BuildProofs(
    ctx context.Context,
    pp  *params.PublicParams,
) (*GnarkTransferTransaction, error) {}
```

### 8.4 Auditing

For Phase 1, auditing reuses the existing SDK auditing framework. Each output token's
`EncryptedNote` is also encrypted for the designated auditor's key (a second encrypted
ciphertext carried in `OutputDescription.AuditCiphertext`). The auditor can decrypt
all note secrets and verify commitments without participating in the proof protocol.

Auditability proofs (the paper's most expensive component ~1,360 ms, 68% of the
original budget) are deferred to Phase 2 to avoid complicating the initial circuit
architecture.

---

## 9. Graph Hiding Extension (Phase 2)

Phase 2 extends every layer to hide the transaction graph. This section specifies
the new primitives and the changes to existing components.

### 9.1 The Conceptual Shift

| Aspect | Phase 1 | Phase 2 |
|--------|---------|---------|
| What a spend reveals | `cm_in` — the exact commitment being spent | `nf`- an opaque nullifier, unlinkable to any commitment |
| Token existence check | `cm_in` lookup in active token set | ZK Merkle membership proof against current tree root |
| Double-spend prevention | `cm_in` in spent set? | `nf` in nullifier set? |
| Ownership check | Ledger: `record.Owner == senderMSPIdentity` | In-circuit: `owner_sk · G == owner_pk` committed inside `cm` |
| Commitment scheme | `Poseidon(v, type, r)` | `Poseidon(v, type, r, owner_pk_u, owner_pk_v)` |
| Ledger state | `active_tokens{cm → owner}`, `spent{cm}` | `commitment_tree` (Merkle root), `nullifier_set{nf}` |

### 9.2 Commitment Accumulator: Incremental Merkle Tree

All token commitments are accumulated in a binary Merkle tree. The root is the single
on-chain summary of the entire token set. Proving a token exists means proving Merkle
membership without revealing which leaf.

Key properties of the tree:
- *Depth 32* — gives 2^32 ≈ 4.29 billion leaf slots
- *Append-only* — tokens are added left-to-right and never removed; spent tokens are tracked only by their nullifier
- *Leaf hash*: Poseidon(domain=1, cm) — domain separation from internal nodes
- *Internal node hash*: Poseidon(domain=0, left, right)
- *Frontier storage* — only the 32 frontier nodes (path from last leaf to root) are stored on-ledger, giving O(depth) storage not O(n)
- *Empty subtree precomputation* — EMPTY[h] = Poseidon(EMPTY[h-1], EMPTY[h-1]) computed once at setup for all 33 depths

Appending a new commitment requires exactly 32 Poseidon evaluations (~0.1 ms). The wallet
stores the leaf position at the time a token is received (from the encrypted note) and
recomputes the current Merkle path from the frontier when spending.

```go
// token/core/zkat-snark-driver/graph/tree/tree.go

// IncrementalMerkleTree is a binary Merkle tree supporting append-only insertions.
// Depth 32 gives 2^32 ≈ 4.29 billion leaf slots.
// Only the 32 frontier nodes are stored on-ledger (O(depth) storage, not O(n)).
type IncrementalMerkleTree struct {
    depth        int               // 32
    frontier     [32]fr.Element    // nodes on path from last leaf to root
    nextPosition uint64            // index of next available leaf slot
    emptyHashes  [33]fr.Element    // precomputed EMPTY[h] for all depths
}
```

### 9.3 Nullifier Scheme

The nullifier is a deterministic one-way function of a token and its position in the
Merkle tree. It is published when a token is spent and added to the nullifier set on-chain.
Double-spending is detected because the same nullifier would appear twice.

```
nf = MixingPedersenHash(cm, pos)
```

MixingPedersenHash is a windowed Pedersen hash over Jubjub. The input is the concatenated
bit representation of cm (254 bits) and pos (64 bits), 318 bits total, split into
106 windows of 3 bits each. Each window contributes a conditionally-negated scalar multiple
of one of 106 fixed Jubjub generator points. The nullifier is the U-coordinate of the
resulting Jubjub point.

The position-dependence is essential. Without it, a previous token holder who knew cm
could monitor the nullifier set for the token's spending. With pos included, they cannot
pre-compute the nullifier unless they also know the leaf position, which is encrypted only
for the current owner in the note ciphertext.

```go
// token/core/zkat-snark-driver/graph/nullifier/nullifier.go

// NullifierHash computes nf = MixingPedersenHash(cm, pos).
// pos is the leaf position in the Merkle tree.
// Returns the U-coordinate of the resulting Jubjub point.
func NullifierHash(cm fr.Element, pos uint64, generators []twistededwards.PointAffine) fr.Element {
    // 1. Bit-decompose cm (254 bits) and pos (64 bits) → 318 bits total
    cmBits  := fieldElementToBits(cm, 254)
    posBits := uint64ToBits(pos, 64)
    bits    := append(cmBits, posBits...)

    // 2. Process in 3-bit windows (106 windows × 3 bits = 318 bits)
    var acc twistededwards.PointAffine
    acc.X.SetZero(); acc.Y.SetOne()  // identity point for twisted Edwards
    curve := twistededwards.GetCurve(twistededwards.BabyJubJub)

    for i := 0; i < len(bits)/3; i++ {
        b0, b1, b2 := bits[3*i], bits[3*i+1], bits[3*i+2]
        sign   := fr.One(); if b2 { sign.Neg(&sign) }
        scalar := fr.Element{}
        scalar.SetUint64(1 + uint64(b0) + 2*uint64(b1))
        scalar.Mul(&scalar, &sign)
        term := curve.ScalarMul(&generators[i], &scalar)
        acc.Add(&acc, &term)
    }

    return acc.X  // U-coordinate is the nullifier
}
```

**Security properties**:

| Property | Guarantee | Basis |
|----------|-----------|-------|
| Uniqueness | Each `(cm, pos)` → unique nullifier | Collision resistance of Pedersen hash over Jubjub |
| Unlinkability | From `nf`, cannot determine `cm` or `pos` | Discrete log hardness on Jubjub |
| Non-malleability | Circuit enforces `nf` from constrained `cm` and `pos` | Shared variable constraint in ExtendedSpendCircuit |
| Position dependence | Same `cm` at different positions → different nullifiers | `pos` is explicit hash input; previous holder cannot predict `nf` |

### 9.4 Extended SpendCircuit

The Phase 2 SpendCircuit proves five compound statements simultaneously. The key
security property is that `cm` is a single shared circuit variable used in groups
2, 3, and 4 since it is impossible to use data from different tokens for different groups.

```go
// token/core/zkat-snark-driver/graph/circuit/extended_spend.go

type ExtendedSpendCircuit struct {
    // ── Public inputs ────────────────────────────────────────
    MerkleRoot   frontend.Variable `gnark:",public"`  // current tree root (anchor)
    Nullifier    frontend.Variable `gnark:",public"`  // nf = MixingPedersenHash(cm, pos)
    ValueCommitX frontend.Variable `gnark:",public"`  // cv Jubjub U-coord
    ValueCommitY frontend.Variable `gnark:",public"`  // cv Jubjub V-coord

    // ── Private inputs ───────────────────────────────────────
    Value          frontend.Variable
    TokenType      frontend.Variable
    Randomness     frontend.Variable
    OwnerSk        frontend.Variable         // spending key
    OwnerPkU       frontend.Variable         // derived in-circuit from OwnerSk
    OwnerPkV       frontend.Variable
    LeafPos        frontend.Variable
    MerklePath     [32]frontend.Variable     // sibling hashes
    PathDirections [32]frontend.Variable     // 0 = current is left child
    RCV            frontend.Variable
}
```

**Constraint budget - ExtendedSpendCircuit**:

| Group | Statement | Constraints |
|-------|-----------|-------------|
| 1 | Ownership: `owner_sk · G == owner_pk` | ~900 |
| 2 | Commitment: `cm = Poseidon(v, type, r, pk_u, pk_v)` | ~380 |
| 3 | Membership: Merkle path (32 levels × ~285) | ~9,120 |
| 4 | Nullifier: `MixingPedersenHash(cm, pos)` | ~2,800 |
| 5 | Value commitment: `cv = v·V + rcv·R` | ~1,800 |
| 6 | Range check | ~130 |
| **Total** | | **~15,130** |

Prove time ~35–45 ms per circuit; still ~44× faster than the paper baseline of 1,993 ms.

### 9.5 Anchor Model

Between proof generation (client) and validation (peer), new tokens may be appended
to the tree, changing the root. The anchor model accepts any root from the last
`ANCHOR_DEPTH` blocks (proposed: 50 blocks, ~100 seconds at 2-second Fabric block
intervals), stored in a circular buffer in the chaincode state.

```go
const AnchorDepth = 50  // configurable per channel

func (v *Verifier) isValidAnchor(root fr.Element) bool {
    for _, r := range v.recentRoots {
        if r.Equal(&root) {
            return true
        }
    }
    return false
}
```

### 9.6 Phase 2 Validator Changes

| Check | Phase 1 | Phase 2 |
|-------|---------|---------|
| Token existence | `cm_in` lookup in active token set | ZK proof: `merkle_root` is a valid anchor |
| Unspent | `cm_in` not in `spent_commitments` | `nf` not in `nullifier_set` |
| Ownership | Ledger `record.Owner == senderMSPIdentity` | Proved in-circuit (groups 1 + 2) |
| State write (spend) | `Del active_tokens[cm_in]`, `Put spent[cm_in]` | `Put nullifier_set[nf]` |
| State write (create) | `Put active_tokens[cm_out]` with owner | Append `cm_out` to Merkle tree; update root in anchor set |

---

## 10. Configuration

### 10.1 Driver Configuration

The gnark driver is configured under the TMS token driver configuration:

```yaml
token:
  tms:
    mytms:
      network: fabric
      channel: mychannel
      namespace: zk
      driver: gnark  # ← selects this driver

      services:
        network:
          fabric:
            # standard Fabric network config

        token:
          driver:
            gnark:
              # Phase (1 = value privacy, 2 = graph hiding)
              phase: 1

              # Setup configuration
              setup:
                # "local" for development, "groth16-ceremony" or "plonk-kzg" for production
                type: local
                # Path to stored proving keys (distributed to clients)
                pkDir: /path/to/proving-keys
                # Public parameters are stored in channel config; path here is for setup only
                ppPath: /path/to/public-params.bin

              # Proof generation tuning
              prover:
                # Number of goroutines for parallel proving (0 = runtime.NumCPU())
                workers: 0
                # Timeout for a single groth16.Prove call
                proveTimeout: 30s

              # Phase 2 specific (ignored if phase = 1)
              graph:
                treeDepth: 32          # 2^32 ≈ 4B token slots
                anchorDepth: 50        # recent roots accepted as valid anchors

              # Benchmarking (development only)
              benchmark:
                enabled: false
                outputPath: /tmp/gnark-bench
```

### 10.2 Configuration Structure

```go
// token/core/zkat-snark-driver/config/config.go

type Config struct {
    Phase  int          // 1 (value privacy) or 2 (graph hiding)
    Setup  SetupConfig
    Prover ProverConfig
    Graph  GraphConfig   // Phase 2 only
}

type SetupConfig struct {
    Type   string  // "local", "groth16-ceremony", "plonk-kzg"
    PKDir  string
    PPPath string
}

type ProverConfig struct {
    Workers      int
    ProveTimeout time.Duration
}

type GraphConfig struct {
    TreeDepth   int
    AnchorDepth int
}
```

---

## 11. Testing Strategy

### 11.1 Unit Tests

**Location**: `*_test.go` alongside each package

**Coverage requirements** (≥ 90% for core circuit and driver logic):

| Package | Key test files |
|---------|---------------|
| `crypto/poseidon` | `TestPoseidonCrossConsistency` — identical outputs from `Hash` and `HashCircuit` |
| `crypto/jubjub` | `TestValueCommitConsistency`, `TestSchnorrSignVerify` |
| `circuit` | `TestSpendCircuitValidWitness`, `TestSpendCircuitInvalidWitness_*`, `TestConstraintCount` |
| `prover` | `TestWitnessBuilder`, `TestOrchestratorParallelism`, `TestBindingSignature` |
| `verifier` | `TestValidatorAcceptsValidTx`, `TestValidatorRejectsInvalidProof`, `TestValidatorRejectsDoubleSpend` |
| `graph/tree` | `TestMerkleAppend`, `TestMerkleRootConsistency` |
| `graph/nullifier` | `TestNullifierCrossConsistency`, `TestNullifierUniqueness` |

**Invalid witness test template** (applied to every circuit):
```go
func TestSpendCircuitInvalidWitness_WrongValue(t *testing.T) {
    // Build a valid assignment for value=100
    assignment := buildValidSpendAssignment(t, 100)
    // Corrupt: set value to 999 (commitment was for 100)
    assignment.Value = 999
    witness, _ := frontend.NewWitness(assignment, ecc.BN254.ScalarField())
    _, err := groth16.Prove(cs, pk, witness)
    require.Error(t, err, "circuit must not accept a witness with wrong value")
    require.Contains(t, err.Error(), "unsatisfied constraint")
}
```

### 11.2 Benchmark Tests

All benchmarks use `go test -bench -benchmem` and report:
- `ns/op`- latency
- `allocs/op`- allocations
- `B/op`- memory

**Benchmark suite**:
```go
// token/core/zkat-snark-driver/bench/bench_test.go

func BenchmarkSpendProve(b *testing.B) {
    // Setup once, benchmark Prove only
}

func BenchmarkOutputProve(b *testing.B) {}

func BenchmarkTransferParallel_2in2out(b *testing.B) {
    // Benchmark the full orchestrator: 2 spend + 2 output proofs in parallel
}

func BenchmarkValidatorVerify_2in2out(b *testing.B) {
    // Benchmark 4 parallel groth16.Verify calls + binding sig + UTXO checks
}

func BenchmarkPoseidon(b *testing.B) {}
func BenchmarkValueCommit(b *testing.B) {}
func BenchmarkNullifierHash(b *testing.B) {}  // Phase 2

func BenchmarkPhase2SpendProve(b *testing.B) {}  // Phase 2 ExtendedSpendCircuit
```

Results are reported against Table 1 of Androulaki et al. (AFT 2020) at the component
level: value conservation, token validity, serial numbers, auditability.

### 11.3 Integration Tests

**Test environment**: local Fabric network using the SDK's existing `make integration-tests-dlog-fabric-t1` pattern, adapted for the gnark driver.

```go
// integration/token/gnark/gnark_test.go

var _ = Describe("gnark Token Driver", func() {
    var (
        network *nwo.Network
        alice   *nwo.Node
        bob     *nwo.Node
        issuer  *nwo.Node
    )

    BeforeEach(func() {
        network = nwo.NewFabricNetwork()
        issuer  = network.AddNode("issuer", nwo.WithIssuer())
        alice   = network.AddNode("alice")
        bob     = network.AddNode("bob")
        network.SetDriver("gnark")
        network.Start()
    })

    It("should issue tokens", func() {
        tx := issuer.Issue("100", "USD", alice)
        Expect(tx).To(BeValid())
        Expect(alice.Balance("USD")).To(Equal("100"))
    })

    It("should transfer tokens", func() {
        issuer.Issue("100", "USD", alice)
        tx := alice.Transfer(bob, "50", "USD")
        Expect(tx).To(BeValid())
        Expect(alice.Balance("USD")).To(Equal("50"))
        Expect(bob.Balance("USD")).To(Equal("50"))
    })

    It("should reject double-spend", func() {
        issuer.Issue("100", "USD", alice)
        alice.Transfer(bob, "100", "USD")
        tx := alice.Transfer(bob, "100", "USD")  // same inputs
        Expect(tx).To(BeInvalid())
        Expect(tx.Message).To(ContainSubstring("already spent"))
    })

    It("should migrate a zkatdlog token", func() {
        // Issue a token using the existing zkatdlog driver
        zkatdlogToken := issuer.IssueWithDriver("zkatdlog", "100", "USD", alice)
        Expect(zkatdlogToken).To(BeValid())

        // Alice migrates it to the gnark driver using MigrationCircuit
        migrationTx := alice.MigrateToken(zkatdlogToken, "gnark")
        Expect(migrationTx).To(BeValid())

        // Alice can now spend the migrated token via the gnark driver
        tx := alice.Transfer(bob, "100", "USD")
        Expect(tx).To(BeValid())
    })

    It("should produce proofs under 50ms", func() {
        issuer.Issue("100", "USD", alice)
        alice.Issue("50", "USD", alice)

        start := time.Now()
        tx := alice.Transfer(bob, "150", "USD")
        elapsed := time.Since(start)

        Expect(tx).To(BeValid())
        Expect(elapsed).To(BeNumerically("<", 50*time.Millisecond))
    })
})
```

---

## 12. Metrics and Monitoring

```go
// token/core/zkat-snark-driver/metrics/metrics.go

type Metrics struct {
    // Proof generation
    ProveLatency      prometheus.Histogram  // per circuit type (spend / output / migration)
    ProveSuccess      prometheus.Counter
    ProveFailure      prometheus.Counter
    WitnessLatency    prometheus.Histogram

    // Proof verification
    VerifyLatency     prometheus.Histogram
    VerifySuccess     prometheus.Counter
    VerifyFailure     prometheus.Counter

    // Binding signature
    BindingSigLatency prometheus.Histogram

    // Full transaction
    TransferLatency   prometheus.Histogram  // prover end-to-end
    ValidatorLatency  prometheus.Histogram  // validator end-to-end

    // Circuit health
    ConstraintCount   prometheus.Gauge      // per circuit; logged at setup time

    // Phase 2 (graph hiding)
    MerkleUpdateLatency  prometheus.Histogram
    NullifierHashLatency prometheus.Histogram

    // Error counters
    InvalidProofCount    prometheus.Counter
    DoubleSpendAttempts  prometheus.Counter
    WitnessEncodingErrors prometheus.Counter
}
```

**Alerting thresholds**:

```yaml
alerts:
  - name: ProveLatencyHigh
    condition: prove_latency_p99 > 100ms
    severity: warning

  - name: VerifyLatencyHigh
    condition: verify_latency_p99 > 10ms
    severity: critical

  - name: InvalidProofSpike
    condition: rate(invalid_proof_count[5m]) > 10
    severity: warning
    # May indicate witness encoding mismatch between driver versions

  - name: DoubleSpendAttempt
    condition: double_spend_attempts_total > 0
    severity: critical
```

---

## 13. Error Handling

### 13.1 Error Taxonomy

```go
// token/core/zkat-snark-driver/errors/errors.go

var (
    // Circuit errors
    ErrUnsatisfiedConstraint = errors.New("circuit constraint not satisfied")
    ErrInvalidWitness        = errors.New("invalid witness")
    ErrProveTimeout          = errors.New("groth16.Prove timed out")

    // Proof verification errors
    ErrInvalidProof          = errors.New("groth16.Verify failed")
    ErrWrongVerificationKey  = errors.New("proof does not match verification key")
    ErrOffCurvePoint         = errors.New("value commitment point not on Jubjub")
    ErrNonCanonicalEncoding  = errors.New("field element is not canonically encoded")

    // Binding signature errors
    ErrInvalidBindingSignature = errors.New("binding signature verification failed")
    ErrConservationViolation   = errors.New("Σv_in ≠ Σv_out")

    // UTXO errors
    ErrTokenNotFound         = errors.New("commitment not found on ledger")
    ErrAlreadySpent          = errors.New("commitment already spent")
    ErrOutputAlreadyExists   = errors.New("output commitment already exists on ledger")
    ErrTokenTypeMismatch     = errors.New("token type mismatch")
    ErrOwnershipViolation    = errors.New("sender does not own this token")

    // Public parameter errors
    ErrInvalidPublicParams   = errors.New("invalid public parameters")
    ErrPublicParamsMismatch  = errors.New("public parameters do not match channel config")
    ErrSetupRequired         = errors.New("trusted setup has not been performed")

    // Phase 2 errors
    ErrNullifierAlreadySpent = errors.New("nullifier already in nullifier set")
    ErrInvalidAnchor         = errors.New("Merkle root is not a valid recent anchor")
    ErrInvalidMerklePath     = errors.New("Merkle path does not verify against root")
)
```

### 13.2 Error Classification and Recovery

| Error | Category | Retry | Action |
|-------|----------|-------|--------|
| `ErrUnsatisfiedConstraint` | Permanent | No | Bug in witness builder; fix code |
| `ErrInvalidProof` | Permanent | No | Reject transaction |
| `ErrNonCanonicalEncoding` | Permanent | No | Reject transaction; check encoding |
| `ErrInvalidBindingSignature` | Permanent | No | Conservation check failed; reject |
| `ErrTokenNotFound` | Permanent | No | Token does not exist; reject |
| `ErrAlreadySpent` | Permanent | No | Double-spend attempt; reject |
| `ErrProveTimeout` | Transient | Yes (1×) | Retry with fresh witness |
| `ErrPublicParamsMismatch` | Config | No | Update public params from channel |
| `ErrInvalidAnchor` (Phase 2) | Transient | Yes (1×) | Refetch Merkle root and reprove |

---

## 14. Security Considerations

### 14.1 Threat Model

**Threat 1: Trusted Setup Compromise**
- **Risk**: If the Groth16 toxic waste survives the setup ceremony, an attacker can generate
  proofs for false statements (e.g., prove they own a token they do not)
- **Mitigation**: Multi-party computation ceremony where at least one participant honestly
  destroys their contribution. For development, a local setup is acceptable. For production,
  a Groth16 MPC ceremony or PLONK/KZG universal SRS (no per-circuit toxic waste) is required.

**Threat 2: Verification Key Substitution**
- **Risk**: If an attacker replaces the verification key in the Fabric channel configuration
  with a key for a modified circuit, fraudulent proofs would be accepted
- **Mitigation**: Verification key updates require the Fabric channel update policy.
  The verification key hash should be logged and monitored; unexpected changes should alert.

**Threat 3: Witness Encoding Mismatch**
- **Risk**: If the prover's WitnessBuilder and the validator's public witness reconstruction
  use different field element encodings, legitimate proofs fail while forged proofs that
  happen to use the validator's encoding might be accepted in degenerate cases
- **Mitigation**: Shared encoding functions in `crypto/encoding.go` used by both components;
  cross-implementation test vectors; `SetBytesCanonical` rejects non-canonical encodings

**Threat 4: Groth16 Proof Malleability**
- **Risk**: Given a valid proof (A, B, C), an adversary can compute (δA, δ⁻¹B, C) that
  also verifies for the same public inputs. If the system uses proof bytes as a unique
  transaction identifier, this enables replay.
- **Mitigation**: Transaction identity is based on the commitment (a public input), not
  the proof bytes. Malleable proofs for the same commitment are rejected at the UTXO check.

**Threat 5: Double-Spend via Concurrent Submission**
- **Risk**: Two transactions spending the same token submitted concurrently may both
  pass cryptographic checks; the ledger check is the last line of defence
- **Mitigation**: Fabric's MVCC conflict detection handles concurrent double-spend at the
  ledger level. The explicit spent-set check provides defence-in-depth for sequential
  double-spends where the MVCC snapshot does not catch the conflict.

**Threat 6: Phase 2: Nullifier Pre-computation**
- **Risk**: A previous token holder who saw the commitment could attempt to monitor
  the nullifier set for the token's spending. A non-position-dependent nullifier
  (`Poseidon(v, type, r)`) would allow this.
- **Mitigation**: Position-dependent nullifier `MixingPedersenHash(cm, pos)` ensures
  that the previous holder, who does not know `pos` (assigned at issuance time, encrypted
  only for the current holder), cannot pre-compute the nullifier.

**Threat 7: Phase 2: Anchor Manipulation**
- **Risk**: If the anchor depth is too large, an old Merkle root could be used to spend
  a token that was since transferred to a different owner, exploiting a stale tree state
- **Mitigation**: The anchor model only accepts roots up to `ANCHOR_DEPTH` blocks old.
  Ownership is proved in-circuit (Phase 2), not via the ledger record, so even a stale
  root anchor cannot be exploited if the prover does not know the current owner's `owner_sk`.

### 14.2 Key Consistency Security Requirement

The following must hold or the entire proof system is insecure:

```
Poseidon.Hash(v, type, r)  [native Go]  ==  PoseidonGadget.Sum(v, type, r)  [gnark circuit]
```

This is verified by a mandatory test suite `TestPoseidonCrossConsistency` that runs
100 random inputs through both implementations and asserts byte-level equality of outputs.
This test must pass before any circuit is used in a non-development environment.

---

## 15. Implementation Phases

### Phase 1: Cryptographic Primitives and Setup (Week 1–2)

- [ ] Package structure under `token/core/zkat-snark-driver/`
- [ ] Native Go Poseidon hash (`crypto/poseidon/hash.go`)
- [ ] gnark Poseidon circuit gadget (`circuit/gadgets/poseidon.go`)
- [ ] Cross-consistency test suite with shared test vectors
- [ ] Jubjub value commitment native Go (`crypto/jubjub/commitment.go`)
- [ ] Jubjub value commitment gnark gadget (`circuit/gadgets/valuecommit.go`)
- [ ] Jubjub Schnorr sign/verify (`crypto/jubjub/schnorr.go`)
- [ ] Public parameters structure and serialization (`pp/pp.go`)
- [ ] Trusted setup utilities (`setup/setup.go`)
- [ ] Driver registration skeleton (`driver/driver.go`)

### Phase 2: Core Circuits (Week 3–4)

- [ ] SpendCircuit R1CS definition and compilation (`circuit/spend.go`)
- [ ] OutputCircuit R1CS definition and compilation (`circuit/output.go`)
- [ ] MigrationCircuit R1CS definition and compilation (`circuit/migration.go`)
- [ ] Valid witness tests for all three circuits
- [ ] Invalid witness tests (wrong value, wrong randomness, wrong cv, zero output)
- [ ] Constraint count logging and budget verification
- [ ] Groth16 setup (local) for all three circuits
- [ ] Benchmark suite: `BenchmarkSpendProve`, `BenchmarkOutputProve`, `BenchmarkValidatorVerify`
- [ ] **Deliverable**: all three circuits with benchmarks confirming > 50× speedup over paper

### Phase 3: Prover and Transaction Assembly (Week 5–6)

- [ ] WitnessBuilder for SpendCircuit and OutputCircuit (`prover/witness.go`)
- [ ] SpendProver and OutputProver with goroutine-safe proving key sharing
- [ ] Binding signature computation
- [ ] Parallel Orchestrator (`prover/orchestrator.go`)
- [ ] Transaction wire format serialization/deserialization
- [ ] Encrypted note construction for recipients
- [ ] Note type, IssueAction, TransferAction (`token/`)
- [ ] Unit tests for prover pipeline

### Phase 4: Validator and Driver Integration (Week 7–8)

- [ ] Chaincode validator: structural checks, public witness reconstruction, parallel verify
- [ ] Binding signature verification
- [ ] UTXO ledger checks (existence, double-spend, type, ownership)
- [ ] State transition: mark spent, add outputs
- [ ] Driver interface implementation (`driver.TokenDriver`, `driver.Validator`, `driver.PublicParametersManager`)
- [ ] WalletService: UTXO selection, key management
- [ ] Configuration loading
- [ ] Unit tests for validator

### Phase 5: Integration Tests and Benchmarks (Week 9–10)

- [ ] Integration test suite: issue, transfer, double-spend rejection, MigrationCircuit
- [ ] Full Fabric network integration test (`integration/token/gnark/`)
- [ ] Achieve ≥ 90% test coverage on core circuit and driver logic
- [ ] Benchmark report comparing each component against Table 1 of Androulaki et al.
- [ ] Documentation PR

### Phase 6: Graph Hiding: Primitives and Circuits (Week 11–12)

- [ ] Incremental Merkle tree with frontier storage (`graph/tree/tree.go`)
- [ ] MixingPedersenHash native Go implementation (`graph/nullifier/nullifier.go`)
- [ ] MixingPedersenHash gnark circuit gadget (`circuit/gadgets/nullifier.go`)
- [ ] Cross-consistency test for nullifier (native Go vs gnark circuit)
- [ ] ExtendedSpendCircuit (all 6 constraint groups) (`graph/circuit/extended_spend.go`)
- [ ] Extended OutputCircuit with owner_pk binding
- [ ] Phase 2 public parameters (NullifierGenerators, TreeDepth)
- [ ] Unit tests and invalid witness tests for extended circuits

### Phase 7: Graph Hiding: Driver Integration (Week 13–14)

- [ ] Phase 2 validator: anchor model, nullifier set checks, Merkle tree updates
- [ ] Phase 2 wallet: position tracking, Merkle path retrieval
- [ ] Phase 2 prover: extended witness builder, path fetching
- [ ] End-to-end integration tests with full graph hiding
- [ ] Performance benchmarks: Phase 1 vs Phase 2 comparison
- [ ] Final PR to `fabric-token-sdk` main repository

---