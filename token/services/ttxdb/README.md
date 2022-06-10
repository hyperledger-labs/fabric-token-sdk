# Token Transactions DB

The Token Transactions DB is a database of audit records. It is used to track the
history of audit events. In particular, it is used to track payments, holdings,
and transactions of any business party identified by a unique enrollment ID.

## Getting Started

Each Token Transactions DB is bound to a single auditor wallet. 
To get the instance of Token Transactions DB bound to a given auditor wallet, use the
following:

```go
   ttxDB := ttxdb.Get(context, wallet)
```

## Append Audit Record

An `Audit Record` can be obtained from a `Token Request`. 
Usually Token Requests are themselves embedded in token transactions.

Here is an example of extraction of an audit record from a token request, and
appending of the record to the Token Transactions DB:

```go
	auditRecord, err := tx.TokenRequest.AuditRecord()
	if err != nil {
		return errors.WithMessagef(err, "failed getting audit records for tx [%s]", tx.ID())
	}
	if err := ttxDB.Append(auditRecord); err != nil {
		return errors.WithMessagef(err, "failed appending audit records for tx [%s]", tx.ID())
	}
```

## Payments

To get a list of payments filtered by given criteria, one must first obtain a `query executor` like
this:

```go
    qe := ttxDB.NewQueryExecutor()
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

The following example shows how to retrieve the total amount of holdings for a given business party,

```go
    filter := qe.NewHoldingsFilter()
    filter, err = filter.ByEnrollmentId(eID).ByType(tokenType).Last(10).Execute()
    if err != nil {
        return errors.WithMessagef(err, "failed getting holdings for enrollment id [%s] and token type [%s]", eID, tokenType)
    }
```

## Transactions

The following example shows how to retrieve the total amount of transactions for a given business party,

```go
    it, err := qe.Transactions(from, to)
	if err != nil {
		return err
    }  
    defer it.Close()

    for {
        tx, err := it.Next()
        if err != nil {
            return err
        }
        if tx == nil {
            break
        }
        fmt.Println(tx)
    }
```