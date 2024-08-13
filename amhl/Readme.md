### Payer

AMHL_INVOICE_FROM_PAYEE

### Intermediary

1. AMHL_TX_FROM_BACK : 
    - Store the transaction details with the given lock
    - Forward the rest to the next node with AMHL_TX_FROM_BACK
    - Partially sign the transactions and send it to prev node (caller) with AMHL_K_P_SIG_FROM_FRONT.
2. AMHL_K_P_SIG_FROM_FRONT:
    - Check the received signature (todo) and store.
    - Generate its partial signature and send it to the next node with AMHL_P_SIG_FROM_BACK
    - Set tx to be locked
3. AMHL_P_SIG_FROM_BACK:
    - Check the received signature (todo) and store.
    - Set tx to be locked

4. AMHL_C_SIG_FROM_FRONT
    - Set the tx to be released.

#### Periodically checks the to-be-released transactions.
    - Verify the signature
    - Extract and generate the complete signature for the previous node
    - Release the lock to the previous node with AMHL_C_SIG_FROM_FRONT

