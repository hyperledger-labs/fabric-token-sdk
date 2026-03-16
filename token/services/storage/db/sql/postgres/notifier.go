/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgxlisten"
)

var logger = logging.MustGetLogger()

var AllOperations = []driver.Operation{driver.Insert, driver.Update, driver.Delete}

// Notifier implements a simple subscription API to listen for updates on a database table.
type Notifier struct {
	table            string
	notifyOperations []driver.Operation
	writeDB          *sql.DB
	listener         *pgxlisten.Listener
	primaryKeys      []primaryKey

	startOnce sync.Once
	closeOnce sync.Once
	ctx       context.Context
	cancel    context.CancelFunc

	// callback
	subscribers []driver.TriggerCallback
	mu          sync.RWMutex
}

var operationMap = map[string]driver.Operation{
	"DELETE": driver.Delete,
	"INSERT": driver.Insert,
	"UPDATE": driver.Update,
}

type primaryKey struct {
	name         driver.ColumnKey
	valueDecoder func(string) (string, error)
}

func NewSimplePrimaryKey(name driver.ColumnKey) *primaryKey {
	return &primaryKey{name: name, valueDecoder: identity}
}

func NewBytePrimaryKey(name driver.ColumnKey) *primaryKey {
	return &primaryKey{name: name, valueDecoder: decodeBYTEA}
}

const (
	payloadConcatenator = "&"
	keySeparator        = "_"
	reconnectInterval   = 10 * time.Second
)

func NewNotifier(writeDB *sql.DB, table, dataSource string, notifyOperations []driver.Operation, primaryKeys ...primaryKey) *Notifier {
	ctx, cancel := context.WithCancel(context.Background())

	n := &Notifier{
		writeDB:          writeDB,
		table:            table,
		notifyOperations: notifyOperations,
		primaryKeys:      primaryKeys,
		listener: &pgxlisten.Listener{
			Connect: func(ctx context.Context) (*pgx.Conn, error) { return pgx.Connect(ctx, dataSource) },
			LogError: func(ctx context.Context, err error) {
				logger.Errorf("error encountered in [%s]: %s", dataSource, err.Error())
			},
			ReconnectDelay: reconnectInterval,
		},
		ctx:    ctx,
		cancel: cancel,
	}

	// attach handler that calls the subscribers
	n.listener.Handle(table, &notificationHandler{
		table:       table,
		primaryKeys: primaryKeys,
		callback:    n.dispatch,
	})

	return n
}

func (db *Notifier) dispatch(operation driver.Operation, m map[driver.ColumnKey]string) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	for _, callback := range db.subscribers {
		callback(operation, m)
	}
}

func (db *Notifier) Subscribe(callback driver.TriggerCallback) error {
	// register the callback
	db.mu.Lock()
	db.subscribers = append(db.subscribers, callback)
	defer db.mu.Unlock()

	// Note that if the db listener is already closed, we still append subscribers here.
	// Clearly, this is not very robust. A better implementation would check if the Notifier is still open, otherwise
	// ignore the Subscribe call or return an error.
	// Since we deprecated this impl, there is no need improve this behavior.

	db.startOnce.Do(func() {
		logger.Debugf("First subscription for notifier of [%s]. Notifier starts listening...", db.table)
		go func() {
			if err := db.listener.Listen(db.ctx); err != nil {
				logger.Errorf("notifier listen for [%s] failed: %s", db.table, err.Error())
			}
		}()
	})

	return nil
}

func (db *Notifier) Close() error {
	db.closeOnce.Do(func() {
		db.cancel() // stop listener goroutine
		db.mu.Lock()
		db.subscribers = nil
		db.mu.Unlock()
	})

	return nil
}

type notificationHandler struct {
	table       string
	primaryKeys []primaryKey
	callback    driver.TriggerCallback
}

func (h *notificationHandler) parsePayload(s string) (driver.Operation, map[driver.ColumnKey]string, error) {
	items := strings.Split(s, payloadConcatenator)
	if len(items) != 2 {
		return driver.Unknown, nil, errors.Errorf("malformed payload: length %d instead of 2: %s", len(items), s)
	}
	operation, values := operationMap[items[0]], strings.Split(items[1], keySeparator)
	if operation == driver.Unknown {
		return driver.Unknown, nil, errors.Errorf("malformed operation [%v]: %s", operation, s)
	}
	if len(values) != len(h.primaryKeys) {
		return driver.Unknown, nil, errors.Errorf("expected %d keys, but got %d: %s", len(h.primaryKeys), len(values), s)
	}
	payload := make(map[driver.ColumnKey]string)
	for i, key := range h.primaryKeys {
		value, err := key.valueDecoder(values[i])
		if err != nil {
			return driver.Unknown, nil, errors.Wrapf(err, "failed to decode value [%s] for key [%s]", values[i], key.name)
		}
		payload[key.name] = value
	}

	return operation, payload, nil
}

func (h *notificationHandler) HandleNotification(ctx context.Context, notification *pgconn.Notification, _ *pgx.Conn) error {
	if notification == nil || len(notification.Payload) == 0 {
		logger.Warnf("nil event received on table [%s], investigate the possible cause", h.table)

		return nil
	}
	logger.DebugfContext(ctx, "new event received on table [%s]: %s", notification.Channel, notification.Payload)
	op, vals, err := h.parsePayload(notification.Payload)
	if err != nil {
		logger.Errorf("failed parsing payload [%s]: %s", notification.Payload, err.Error())

		return errors.Wrapf(err, "failed parsing payload [%s]", notification.Payload)
	}
	h.callback(op, vals)

	return nil
}

func (db *Notifier) UnsubscribeAll() error {
	logger.Debugf("Unsubscribe called")

	// unregister all callbacks
	db.mu.Lock()
	defer db.mu.Unlock()
	clear(db.subscribers)

	return nil
}

func (db *Notifier) GetSchema() string {
	primaryKeys := make([]driver.ColumnKey, len(db.primaryKeys))
	for i, key := range db.primaryKeys {
		primaryKeys[i] = key.name
	}
	funcName := triggerFuncName(primaryKeys)
	lock := createLockTag(funcName)

	return fmt.Sprintf(`
	SELECT pg_advisory_xact_lock(%d);
	CREATE OR REPLACE FUNCTION %s() RETURNS TRIGGER AS $$
			DECLARE
			row RECORD;
			output TEXT;

			BEGIN

			-- Checking the Operation Type
			IF (TG_OP = 'DELETE') THEN
				row = OLD;
			ELSE
				row = NEW;
			END IF;
			
			-- Forming the Output as notification. You can choose you own notification.
			output = TG_OP || '%s' || %s;
			
			-- Calling the pg_notify for my_table_update event with output as payload
	
			PERFORM pg_notify('%s',output);
			
			-- Returning null because it is an after trigger.
			RETURN NULL;
			END;
	$$ LANGUAGE plpgsql;
	
	CREATE OR REPLACE TRIGGER trigger_%s
	AFTER %s ON %s
	FOR EACH ROW EXECUTE PROCEDURE %s();
	`,
		lock,
		funcName,
		payloadConcatenator, concatenateIDs(primaryKeys),
		db.table,
		db.table,
		convertOperations(db.notifyOperations), db.table,
		funcName,
	)
}

func (db *Notifier) CreateSchema() error {
	return common.InitSchema(db.writeDB, db.GetSchema())
}

func convertOperations(ops []driver.Operation) string {
	opMap := collections.InverseMap(operationMap)
	opStrings := make([]string, len(ops))
	for i, op := range ops {
		opString, ok := opMap[op]
		if !ok {
			panic("op " + strconv.Itoa(int(op)) + " not found")
		}
		opStrings[i] = opString
	}

	return strings.Join(opStrings, " OR ")
}

func triggerFuncName(keys []string) string {
	return "notify_by_" + strings.Join(keys, "_")
}

func concatenateIDs(keys []string) string {
	fields := make([]string, len(keys))
	for i, key := range keys {
		fields[i] = "row." + key
	}

	return strings.Join(fields, fmt.Sprintf(" || '%s' || ", keySeparator))
}

func createLockTag(m string) uint64 {
	h := sha256.Sum256([]byte(m))

	return binary.BigEndian.Uint64(h[:])
}
