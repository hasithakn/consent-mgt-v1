/*
 * Copyright (c) 2025, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package model

import (
	"errors"
	"fmt"
)

// ExecuteTransaction executes multiple queries in a single atomic transaction.
// If any query fails, all changes are rolled back.
//
// Example usage:
//
//	queries := []func(tx TxInterface) error{
//	    func(tx TxInterface) error {
//	        _, err := tx.Exec("INSERT INTO users (id, name) VALUES (?, ?)", id, name)
//	        return err
//	    },
//	    func(tx TxInterface) error {
//	        _, err := tx.Exec("UPDATE accounts SET balance = ? WHERE user_id = ?", balance, id)
//	        return err
//	    },
//	}
//	err := ExecuteTransaction(db, queries)
func ExecuteTransaction(db DBInterface, queries []func(tx TxInterface) error) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Wrap tx in TxInterface
	txInterface := NewTx(tx)

	// Execute each query
	for i, query := range queries {
		if err := query(txInterface); err != nil {
			// Rollback on any error
			if rollbackErr := txInterface.Rollback(); rollbackErr != nil {
				// Combine both errors
				return errors.Join(
					fmt.Errorf("query %d failed: %w", i, err),
					fmt.Errorf("rollback failed: %w", rollbackErr),
				)
			}
			return fmt.Errorf("query %d failed: %w", i, err)
		}
	}

	// Commit if all queries succeed
	if err := txInterface.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
