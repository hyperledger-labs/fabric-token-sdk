# Token Transactions DB

The Token Transactions DB is a database of audit records. It is used to track the
history of audit events. In particular, it is used to track payments, holdings,
and transactions of any business party identified by a unique enrollment ID.

## Getting Started

Each Token Transactions DB is bound to a wallet to be uniquely identifiable. 
To get the instance of the Token Transactions DB bound to a given wallet, 
use the following:

```go
   ttxDB := ttxdb.Get(context, wallet)
```

## Append Token Requests

A Token Request describes a set of token operations in a backend agnostic language.
A Token Request can be assembled directly using the Token API or by using service packages like 
[`ttx`](https://github.com/hyperledger-labs/fabric-token-sdk/tree/main/token/services/ttx).

Once a Token Request is assembled, it can be appended to the Token Transactions DB as follows:

```go
	if err := ttxDB.Append(tokenRequest); err != nil {
		return errors.WithMessagef(err, "failed appending audit records for tx [%s]", tx.ID())
	}
```

It is also possible to append just the transaction records corresponding to a given Token Request as follows:

```go
	if err := ttxDB.AppendTransactionRecord(tokenRequest); err != nil {
		return errors.WithMessagef(err, "failed appending audit records for tx [%s]", tx.ID())
	}
```

## Payments

To get a list of payments filtered by given criteria, one must first obtain a `query executor` like
this:

```go
    qe := ttxDB.NewQueryExecutor()
    defer aqe.Done() // Don't forget to close the query executor
```

Now, we are ready to perform payment queries. 
The following example shows how to retrieve the total amount of last 10 payments made by a given 
business party, identified by the corresponding enrollment ID, for a given token type.

```go
    filter := qe.NewPaymentsFilter()
    filter, err = filter.ByEnrollmentId(eID).ByType(tokenType).Last(10).Execute()
    if err != nil {
        return errors.WithMessagef(err, "failed getting payments for enrollment id [%s] and token type [%s]", eID, tokenType)
    }
    sumLastPayments := filter.Sum()
```

## Holdings

The following example shows how to retrieve the current holding of a given token type for a given business party.
Recall that the current holding is the sum of all inbound and outbound payments.

```go
    filter := qe.NewHoldingsFilter()
    filter, err = filter.ByEnrollmentId(eID).ByType(tokenType).Execute()
    if err != nil {
        return errors.WithMessagef(err, "failed getting holdings for enrollment id [%s] and token type [%s]", eID, tokenType)
    }
```

## Transaction Records

The following example shows how to retrieve the total amount of transactions for a given business party,

```go
	it, err := qe.Transactions(ttxdb.QueryTransactionsParams{From: p.From, To: p.To})
	if err != nil {
		return errors.WithMessagef(err, "failed getting transactions for enrollment id [%s]", eID)
	}
	defer it.Close()

	for {
		tx, err := it.Next()
		if err != nil {
			return errors.WithMessagef(err, "failed getting transactions for enrollment id [%s]", eID)
        }
		if tx == nil {
			break
		}
		fmt.Printf("Transaction: %s\n", tx.ID())
	}
```