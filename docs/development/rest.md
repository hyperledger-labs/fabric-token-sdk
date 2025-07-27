# REST API Development Guidelines

## REST API definitions

At the beginning of the development of a new feature (e.g. issuance service, payment service) there should be a proper definition of the REST API triggering the initiating view. This API definition serves multiple purposes.

- Specify external inputs required to run the view including their types, required fields, etc.
- Specify expected responses from the view including their types, expected fields, and response codes.
- Specify required headers, e.g. Bearer tokens for authentication.
- Allow testers to design and verify test cases for the developed features.
- Allow Software developers to develop applications on top of the API specification.

More specifically, the API definition should include
- Request definition
    - URL / rest endpoints
    - Header definitions (Auth token, Content-Type (e.g. application/JSON))
    - Payload specification (Mandatory/optional fields)
    - Paramter specification (filter attributes etc.)
    - (optional) Min/Max values to be enforced
    - (optional) Min/Max string length, etc.
- Response definition
    - JSON spec (required response fields etc.)
    - Return codes including Error Codes
    - Returned error messages (for advanced testing)
    - Boundary conditions
    - Max/Min values
    - Input parameter conventions (special character support etc.)



A good way to specify the API is to leverage the Swagger Editor (https://editor-next.swagger.io/) to create the Swagger API.

The REST API responses can be found under https://restfulapi.net/http-status-codes/.

Attached is a simplified example of a specification of a cash token issuance API:

```yaml
openapi: 3.0.3
info:
  title: Digital Currency API
  version: 1.0.0
paths:
  /currency/issue:
    post:
      tags:
        - currency
      summary: Issue new currency token
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NewToken'
            example: 
              tokenType: USDCoin
              amount: 100
              issuerWalletId: wid001
      responses:
        '201':
          description: Token issued successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Token'
              example:
                transactionId: 2db46c0bbbfacfbbf0e7433e5c0cd66860949cdefd9c8654d4d71c3517bc340c
        '400':
          description: Bad Request
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
              example:
                error: "Invalid input"
                code: 400
                details: "Required inputs missing."
        '500':
          description: Server error
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
              example:
                error: "Internal Server Error"
                code: 500
                details: "Something went wrong on the server."
      security:
        - ApiKeyAuth: []

components:
  schemas:
    NewToken:
      type: object
      properties:
        tokenType:
          type: string
        amount:
          type: integer
        issuerWalletId:
          type: string
    Token:
      type: object
      properties:
        transactionId:
          type: string
    ErrorResponse:
      type: object
      properties:
        error:
          type: string
        code:
          type: integer
        details:
          type: string
securityDefinitions:
  ApiKeyAuth:
    in: header
    name: Authorization
    type: apiKey
```

## REST API naming conventions

It is important to follow common guidelines for the naming conventions for REST endpoints. There are very good references that describe best practices for REST endpoint, see 
 - https://medium.com/@nadinCodeHat/rest-api-naming-conventions-and-best-practices-1c4e781eb6a5
 - https://restfulapi.net/resource-naming/

 The main concept in REST is a `resource`. There may be collections of resources, e.g. `/wallets` or singleton resources, e.g. `/wallet/{id}`.

 There are many recommendations in the references above, here is a list of the most important ones:

- Use nouns for resources, not verbs
  - GOOD: GET `/wallets/{id}/balance` 
  - BAD: GET `/getWallet/{id}/getBalance`
 
- Use plural for all resource collections
  - GOOD: GET `/wallets/{id}`
  - BAD: GET `/wallet/{id}` 

- Use hyphens instead of `CamelCase` or `under_scores`
  - GOOD: `/asset-types/`

- Don't use CRUD function names in URIs
  - GOOD: POST `/wallets`: create new wallet using data in `body`
  - GOOD: GET `/wallets`: get information for all wallets
  - GOOD: GET `/wallets/{id}`: get information for wallet with `id`
  - BAD: GET `/get-wallets/{id}`

- Use query component / parameters for pagination/filtering/sorting
  - GOOD: GET `/wallets?limit=4&offset=8`

## REST API initial endpoint proposal

Disclaimer: This is an initial list of proposed endpoints as a recommended starting point and basis for further discussion.
It's neither complete nor fully verified at this point.

However, it should be considered as a preferred way to structure the REST endpoints given the general guidelines described above. 

### User

#### wallets

| Request  | Endpoint | Function |
|----------|----------|----------|
| POST     | /wallets | Create new wallet for logged in user |
| GET      | /wallets | Get list of wallet info for all wallets of the logged in user |
| GET      | /wallets/{wallet-id} | Get wallet info for wallet with `wallet-id` | 
| PUT      | /wallets/{wallet-id} | Update wallet info for wallet with `wallet-id` | 
| DELETE   | /wallets/{wallet-id} | Delete wallet with `wallet-id` (if allowed) | 
| GET      | /wallets/{wallet-id}/balance | Get balance for wallet with `wallet-id` | 

#### assets

| Request  | Endpoint | Function |
|----------|----------|----------|
| POST     | /assets/currencies/transfers | Transfer currency from logged-in user to recipient |
| GET      | /assets/currencies/transfers | Get list of all transfers of the logged in user |
| GET      | /assets/currencies/transfers/{tx-id} | Get transfer details for `tx-id` |
| POST     | /assets/currencies/withdrawals | Transfer currency from issuer to logged-in user |
| POST     | /assets/currencies/redemptions   | Transfer currency from logged-in user to issuer|


| Request  | Endpoint | Function |
|----------|----------|----------|
| POST     | /assets/bonds/transfers | Transfer bonds from logged-in user to recipient |
| GET      | /assets/bonds/transfers | Get list of all transfers of the logged in user |
| GET      | /assets/bonds/transfers/{tx-id} | Get transfer details for `tx-id` |
| POST     | /assets/bonds/withdrawals | Transfer bonds from issuer to logged-in user |
| POST     | /assets/bonds/redemptions   | Transfer bonds from logged-in user to issuer|


### Issuer

#### wallets

| Request  | Endpoint | Function |
|----------|----------|----------|
| POST     | /wallets | Create new wallet for logged in issuer |
| GET      | /wallets | Get wallet info for all wallets of the logged in issuer |
| GET      | /wallets/{wallet-id} | Get wallet info for wallet with `wallet-id` | 
| PUT      | /wallets/{wallet-id} | Update wallet info for wallet with `wallet-id` | 
| DELETE   | /wallets/{wallet-id} | Delete wallet with `wallet-id` (if allowed) | 


#### assets

| Request  | Endpoint | Function |
|----------|----------|----------|
| POST     | /assets/currencies | Create new asset-type (speciied in body) |
| GET      | /assets/currencies | Get list of all asset-types of the logged in issuer |
| GET      | /assets/currencies/{asset-id} | Get details of asset-type with `asset-id` |
| POST     | /assets/currencies/issues | Mint currency of an asset-type specified in the body |
| GET      | /assets/currencies/issues | Get list of all issuing operations of the issuer |
| GET      | /assets/currencies/issues/{tx-id} | Get issue details for `tx-id` |
| POST     | /assets/currencies/redemptions | Burn currency of an asset-type specified in the body|
| GET      | /assets/currencies/redemptions | Get list of all redemption operations of the issuer |
| GET      | /assets/currencies/redemptions/{tx-id} | Get redemption details for `tx-id` |
| POST     | /assets/currencies/transfers | Transfer currency from issuer to recipient |
| GET      | /assets/currencies/transfers | Get list of all transfers of the issuer |
| GET      | /assets/currencies/transfers/{tx-id} | Get transfer details for `tx-id` |

| Request  | Endpoint | Function |
|----------|----------|----------|
| POST     | /assets/bonds | Create new bond asset-type (speciied in body) |
| GET      | /assets/bonds | Get list of all asset-types of the logged in issuer |
| GET      | /assets/bonds/{asset-id} | Get details of asset-type with `asset-id` |
| POST     | /assets/bonds/issues | Mint currency of an asset-type specified in the body |
| GET      | /assets/bonds/issues | Get list of all issuing operations of the issuer |
| GET      | /assets/bonds/issues/{tx-id} | Get issue details for `tx-id` |
| POST     | /assets/bonds/redemptions | Burn currency of an asset-type specified in the body|
| GET      | /assets/bonds/redemptions | Get list of all redemption operations of the issuer |
| GET      | /assets/bonds/redemptions/{tx-id} | Get redemption details for `tx-id` |
| POST     | /assets/bonds/transfers | Transfer currency from issuer to recipient |
| GET      | /assets/bonds/transfers | Get list of all transfers of the issuer |
| GET      | /assets/bonds/transfers/{tx-id} | Get transfer details for `tx-id` |

### Auditor

#### assets

| Request  | Endpoint | Function |
|----------|----------|----------|
| GET      | /assets/currencies/transfers | Get list of all transfers of the logged in user |
| GET      | /assets/currencies/transfers/{tx-id} | Get transfer details for `tx-id` |



| Request  | Endpoint | Function |
|----------|----------|----------|
| GET      | /assets/bonds/transfers | Get list of all transfers of the logged in user |
| GET      | /assets/bonds/transfers/{tx-id} | Get transfer details for `tx-id` |

