# Implementation Plan - Identity Service Observability

Implement metrics instrumentation and a lightweight circuit breaker for the identity service.

## Goal Description
The goal is to enhance the `identity.Provider` with observability features (metrics) and a circuit breaker to handle potential overloads or failures gracefully.

## Implementation Steps
1. [x] Define `IdentityMetrics` struct and initialization in `token/services/identity/metrics.go`.
2. [x] Implement `CircuitBreaker` and `CircuitBreakerConfig` in `token/services/identity/cb.go`.
3. [x] Integrate metrics and circuit breaker into `identity.Provider` in `token/services/identity/provider.go`.
4. [x] Wrap public entry points with metrics collection and circuit breaker logic.
5. [x] Add unit tests for metrics and circuit breaker in `token/services/identity/observability_test.go`.
6. [x] Enhance tests to verify actual metrics increments.
7. [x] Run all tests in `token/services/identity` to ensure no regressions.
8. [x] Perform final code cleanup and formatting.

## Implementation Progress
- [x] Step 1-8: Completed.

## Notes & Decisions
- The implementation covers all required entry points from the `driver.IdentityProvider` interface.
- Methods that do not return errors (`AreMe`, `IsMe`, `RollbackPartialRecipientRegistration`) have metrics collection but no circuit breaker rejection.
- Metrics collection is performed at the very beginning of each method to capture all requests, including those rejected by the circuit breaker.
- Safe defaults for the circuit breaker (Threshold: 5, Cooldown: 30s) are implemented in `NewCircuitBreaker`.

✅ COMPLETE
