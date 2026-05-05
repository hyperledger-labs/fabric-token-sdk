# Versioning for TTX Interactive Protocols — Design Exploration

> **Status:** Exploration / proposal — draft for maintainer review (issue #1654).
> **Scope:** All interactive views in `token/services/ttx/` that exchange typed
> messages over FSC sessions. Output of this document is a proposal; implementation
> is deliberately deferred until after maintainer alignment.

---

## 1. Problem

The interactive views in `token/services/ttx/` exchange messages over FSC sessions
using `session.Send` / `session.Receive`. There is no version or capability marker
on the wire. If two nodes run different versions of the token-sdk, they connect a
session and "talk" — JSON silently ignores unknown fields and tolerates missing
optional ones — but they may misinterpret each other's intent with **no error and
no diagnostic**.

As recipient flows gain new fields (slimmer responses, future nonce/signature
attestations, ack-only response modes), and as other protocols evolve in similar
ways, this gap becomes a real source of silent failures. The same concern applies
across the package, not just to recipient views.

---

## 2. Surface Map

The table below catalogues every interactive view in `token/services/ttx/` that
exchanges a typed message over a session.

### 2.1 JSON-typed message exchanges

These flows marshal Go structs over a JSON session. They are the primary subjects
of versioning concern, since struct shape evolution is exactly where silent
misinterpretation can occur.

| File | Initiator view | Responder view | Wire structs |
|------|----------------|----------------|--------------|
| `recipients.go` | `RequestRecipientIdentityView` | `RespondRequestRecipientIdentityView` | `RecipientRequest` → `RecipientData` (+ optional `MultisigRecipientData` follow-up) |
| `recipients.go` | `ExchangeRecipientIdentitiesView` | `RespondExchangeRecipientIdentitiesView` | `ExchangeRecipientRequest` → `RecipientData` |
| `withdrawal.go` | `RequestWithdrawalView` | `ReceiveWithdrawalRequestView` | `WithdrawalRequest` (one-way) |
| `upgrade.go` | `UpgradeTokensInitiatorView` | `UpgradeTokensResponderView` | `UpgradeTokensAgreement` ↔ `UpgradeTokensRequest` |

### 2.2 Raw / opaque-byte exchanges

These flows already embed protocol-level structure inside their payloads
(`token.Request`, signatures). The wire shape from a session perspective is just
`[]byte`; versioning lives inside the inner payload and is **out of scope** for
this proposal.

| File | View(s) | Payload |
|------|---------|---------|
| `auditor.go` | `AuditingViewInitiator` | raw `tx.Bytes()` |
| `endorse.go` | `EndorseView` | raw signature bytes |
| `accept.go` | `AcceptView` | raw ack signature bytes |
| `collectactions.go` | `collectActionsView` | raw tx bytes, `Actions`, `ActionTransfer` |
| `collectendorsements.go` | `CollectEndorsementsView` | `signatureRequestRaw`, `txRaw`, raw signatures |

### 2.3 Observed evolution patterns (why versioning matters)

- `RecipientData` gained `TokenMetadataAuditInfo` after the original 3-field
  shape; older peers serializing the original shape are silently tolerated.
- `WithdrawalRequest` and `UpgradeTokensRequest` both embed `RecipientData`
  inline, so any change to the recipient shape transitively touches them.
- The slim-ack work in [#1652](https://github.com/hyperledger-labs/fabric-token-sdk/issues/1652) is the first deliberate change to a *response*
  shape: an old initiator receiving a slim ack from a new responder would
  re-register with empty `AuditInfo` / metadata, producing wrong data without
  any error.

---

## 3. Approach Evaluation

Four approaches were considered. Each is evaluated against five criteria:

- **Complexity** (lines of new infrastructure)
- **Blast radius** (how many existing call sites change)
- **Author ergonomics** (is it easy to do the right thing?)
- **Mixed-version handling** (what happens between old and new peers?)
- **Coexistence with [#1622](https://github.com/hyperledger-labs/fabric-token-sdk/issues/1622)** (does this conflict with proto-direction?)

### 3.1 Embedded version field

Add a `Version uint32` (or `Capabilities []string`) field to each top-level
request struct (`RecipientRequest`, `ExchangeRecipientRequest`,
`WithdrawalRequest`, `UpgradeTokensRequest`, etc.). Responders branch on the
version and choose how to respond.

**Pros**
- Smallest infrastructure change; no new types, no envelope wrapper.
- Each view owns its own evolution explicitly.

**Cons**
- Repetitive: the same versioning logic repeats in every view.
- Asymmetric: response messages (`RecipientData`) have no obvious place to
  carry a version, since they are the `RecipientData` defined in `token/driver/`.
- Old peers won't know to *send* a version, so detection of unversioned
  messages still falls back to "treat as v0". No improvement on the silent
  failure that already happens when fields are missing.

### 3.2 Envelope wrapper (`{ Version, Body }`)

Wrap every interactive message in a thin envelope handled at the session helper
layer in `token/services/utils/json/session/json.go`. Concretely:

```go
type Envelope struct {
    Version uint32
    Type    string  // optional discriminator, e.g. "RecipientRequest"
    Body    json.RawMessage
}
```

The session helpers (`Send` / `Receive`) marshal the body inside the envelope
and unwrap on receive.

**Pros**
- One central place for versioning; views stay shape-focused.
- Receiver can detect version mismatch *before* attempting to decode the body —
  enables clean fail-fast diagnostics.
- Wraps both requests and responses uniformly, solving the asymmetry of 3.1.
- Compatible with existing `session.Marshaller` abstraction.

**Cons**
- Larger initial change to the session layer. All views adopt simultaneously
  (or coexist with raw-mode senders during a transition).
- Old peers cannot read the envelope at all — pure breaking change unless
  paired with a fallback (see §4).

### 3.3 Negotiation handshake

Add a one-shot capability exchange at session start: each side announces its
supported protocol version range and they pick the highest common version
before exchanging any business message.

**Pros**
- Most flexible — supports forward and backward compatibility cleanly.
- Capability-based handshake (rather than monotonic version) lets views ship
  optional features (e.g., nonce/signature) selectively.

**Cons**
- Largest infrastructure cost — requires session-establishment hooks in FSC
  that don't exist today.
- Adds a round-trip to every interactive flow, which matters for short flows
  like recipient identity request.
- Most divergent from current architecture; highest reviewer burden.

### 3.4 Alignment with #1622 (proto-style envelopes)

Issue [#1622](https://github.com/hyperledger-labs/fabric-token-sdk/issues/1622) explores moving on-wire types to proto-defined envelopes. If the
direction is "all interactive ttx messages become proto messages with a versioned
envelope," that subsumes the JSON-session versioning question — proto handles
field-level forward/backward compatibility natively, and a top-level
`oneof`-discriminator can carry message type.

**Pros**
- Single direction across the codebase; no parallel JSON-versioning machinery.
- Field-level compat for free (proto adding/removing fields is well-defined).

**Cons**
- Migration cost is large and out of scope for a quick wire-hardening fix.
- Couples the (smaller) versioning question to the (larger) serialization
  migration.
- Schedule: [#1622](https://github.com/hyperledger-labs/fabric-token-sdk/issues/1622) may land in a different cycle from M3 wire-slimming work.

---

## 4. Recommendation — Minimal v1

**Adopt Approach 3.2 (envelope wrapper) with a JSON-compatible header field,
applied at the session helper layer, scoped initially to the four JSON-typed
flows in §2.1.**

### 4.1 Wire shape

```go
// token/services/utils/json/session/envelope.go (new)
type Envelope struct {
    Version uint32          `json:"v"`           // monotonic protocol version
    Type    string          `json:"t,omitempty"` // optional message-type tag
    Body    json.RawMessage `json:"b"`
}
```

### 4.2 Why this approach

- **Single point of change.** All future struct evolutions are governed by the
  envelope, not by ad-hoc per-view checks.
- **Fail-fast.** Receiver inspects `Version` first; mismatches produce a
  clean "unsupported protocol version v=N (supported: 1..2)" error rather
  than silent miscoercion.
- **Symmetry.** Both requests and responses pass through the same envelope —
  no awkward asymmetry as in 3.1.
- **Doesn't preclude 3.4.** When proto envelopes from #1622 land, the JSON
  envelope can be replaced with the proto envelope at the same single layer.
  In the meantime, work on M3 doesn't have to wait.

### 4.3 What v1 ships

- A new `Envelope` type in `token/services/utils/json/session/`.
- Updated `Send` / `Receive` helpers that wrap/unwrap automatically when the
  registered marshaller is the JSON one.
- `Version = 1` baseline on every existing JSON flow listed in §2.1.
- A clear error type (`session.ErrUnsupportedVersion`) on receive-side
  mismatch.

### 4.4 What v1 does **not** ship

- **No backward compatibility shim.** Per @adecaro's guidance on #1620,
  retro-compatibility is intentionally deferred. The output of this exploration
  is the place where that policy should be re-decided (see §5).
- **No capability negotiation.** Monotonic version is sufficient for v1.
  Capability bits can be added later within the envelope without breaking
  the wire shape.
- **No automatic migration of raw-byte flows in §2.2.** Those have their own
  evolution stories.

---

## 5. Mixed-Version Behavior

This section frames the policy decision that adecaro asked us to defer to this
exploration (#1620, Q3 answer).

Three plausible policies, in order of conservatism:

### 5.1 Strict — reject unversioned messages

A v1 peer receiving an unversioned message (no envelope) treats it as a
protocol error and returns. Old peers cannot talk to new peers.

- **Pros:** clean, predictable, no silent paths.
- **Cons:** hard cutover; deployment must be coordinated.

### 5.2 Tolerant fallback — accept legacy on first version

A v1 peer attempts envelope decoding; on failure, falls back to legacy
(unwrapped) decoding for one version. This is a deprecation window of one
release.

- **Pros:** smooth transition for one release cycle.
- **Cons:** requires the dual-decode path; small risk of mistaking a malformed
  envelope for a legacy message.

### 5.3 Capability-tagged (forward-only)

Each side advertises support during the first handshake; if neither side has v1,
both fall back to legacy decoding. After v2, only enveloped traffic is supported.

- **Pros:** explicit and observable; longest deprecation window.
- **Cons:** highest infrastructure cost (a real handshake — see §3.3).

**Recommendation:** start with **5.1 (strict)** for v1, scoped to a single
release where the wire shape is announced loudly in the changelog. Move to
5.2-style fallback only if real-world deployment data shows the strict policy
is too aggressive.

---

## 6. Coordination with #1622

The proposed envelope is **non-conflicting** with #1622's proto direction:

- If #1622 lands first, the JSON envelope can be skipped entirely; views adopt
  the proto envelope directly.
- If this v1 lands first, #1622's migration replaces the JSON envelope at the
  session helper layer — views need no further changes since they already
  delegate version handling to the helper.
- Both approaches put versioning at the same architectural layer (the session
  helper), so the decision of *which* envelope is implementable is decoupled
  from the *adoption pattern* (which is the same).

This makes the v1 envelope a low-regret intermediate step.

---

## 7. Migration Path (Sketch)

```
Step 1: introduce Envelope type + helper changes (no behavior change yet,
        all existing flows still send legacy shape behind a feature-gate)
Step 2: switch RecipientRequest / RecipientData flows to envelope-mode (v1)
Step 3: switch WithdrawalRequest, UpgradeTokens* flows to envelope-mode (v1)
Step 4: enable receive-side strict-mode (§5.1)
Step 5: bump v2 when the next field-level change ships (e.g., nonce/signature)
```
---

## 8. Open Questions for Maintainer Review

1. **Mixed-version policy** (§5) — strict, tolerant fallback, or capability
   handshake? Recommendation is strict for v1; want explicit confirmation.
2. **Envelope field naming** (§4.1) — `v/t/b` (terse) vs `version/type/body`
   (verbose). Terse is friendlier for hand-debugging payloads but may surprise
   readers. No strong preference.
3. **Type discriminator** (§4.1) — keep `Type` optional, mandatory, or omit
   entirely? Optional gives runtime sanity checks; mandatory adds rigidity;
   omitting reduces complexity. Recommendation: optional and unused at v1,
   reserved for future capability routing.
4. **Coordinate with #1622 owner** — should this exploration land first as a
   stop-gap, or wait for the proto direction to settle? Recommendation: land
   first, with a clear note that the envelope is the migration seam.
5. **Scope expansion** — should the §2.2 raw-byte flows eventually adopt the
   same envelope, or keep them on their existing inner-payload versioning?
   Out of scope for v1, worth a follow-up issue.

---