# Test Flows
## Flow 1: request & transfer

1. User starts with zero funds
2. User requests money indicated in `funds` within multiple transactions
3. User transfers money to accounts indicated in `payTo` until he spends all requested money in step 1

Transaction generator will execute flow 1 in parallel for multiple users.
Money issuance and payment transactions will be executed one after another with a `delay` indicated in configuration

Size of each transaction is determined by `payment` configuration and corresponding strategy.

## Flow 2: request & loop transfer
1. User starts with zero funds
2. User requests money indicated in `funds` within multiple transactions
3. User transfers money to accounts indicated in `payTo` until he has enough funds. If he doesn't have enough funds, he waits until he receives money or skips iteration after a timeout

## Success criteria

Test is successful if:
1. Total funds in circulation at the end of the test is equal to total requested funds
2. All transactions have been executed successfully with no more than expected latency
3. Intermediary returns same balance per user which is expected by tx generator


## Configuration setup

```yaml
app:
  logging: INFO
userProvider:
  users:
    - name: alice
      username: alice
      password: alicePWD
      endpoint: http://9.12.248.105:8081
    - name: bob
      username: bob
      password: bobPWD
      endpoint: http://9.12.248.105:8082
    - name: charlie
      username: charlie
      password: charliePWD
      endpoint: http://9.12.248.105:8081
    - name: dave
      username: dave
      password: davePWD
      endpoint: http://9.12.248.105:8082
  httpClient:
    timeout: 1s                 # connection timeout
    maxConnsPerHost: 0          # 0 - means no limit
    maxIdleConnsPerHost: 100
intermediary:
  delayAfterInitiation: 250ms   # delay after receiver has registered transaction. If "delayAfterInitiation <= 0" no delay will happen   
suites:
  - name: Concurrent payments   # Suite run sequentially, one after another
    iterations: 100             # how many times this suite will be executed
    delay: 100ms                # delay between transactions within one test case
    delayAfterInitiation: 250 
    poolSize: 2                 # number of parallel requests to server from one test case
    useExistingFunds: true      # tells to generator to use all existing user money within every iteration
    cases:                      # Cases run in parallel, one after another
      - name: Alice to all
        payer: alice            # Refers to users.name
        payees:                 # Refers to users.name
          - bob
          - charlie
          - dave
        issue:                  # We will issue using the distribution until we reach the total. Then we start transferring
          total: 20
          distribution: const:1 # options: "const:1" | "const:1:2:3" | "uniform:3:10"
          execute: true         # if false, this step will be omitted
        transfer:
          distribution: const:1:2 # when budget is 11, will generate "1,2,1,2,1,2,1,1"
          execute: true         # if false, this step will be omitted
      - name: Bob to all
        payer: bob
        payees:
          - charlie
          - dave
        issue:
          total: 20
          distribution: uniform:3:10 # if budget is 20, may generate: "4,7,3,5,1", last value can be smaller then smmalest value in distribution indicated
          execute: true
        transfer:
          distribution: const:1
          execute: true
```

## Collected metrics

1. Transaction average time (TG)
2. Min, max time of withdrawal and payment (TG)
3. How much money users paid / received to cross compare with the balance in the end
4. RAM & CPU (TG, I) // TODO


TG - transaction generator, I - intermediary

# How to run

To build application
```
go build -o generator
```

To run application with custom configuration file:
```
CBDC_E2E_TX_CONFIG_FILE=./config.yaml ./generator
```
