package com.banking.service;

import com.banking.model.Account;
import com.banking.model.Transaction;
import com.banking.repository.AccountRepository;
import com.banking.repository.TransactionRepository;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.math.BigDecimal;
import java.util.Collections;
import java.util.List;
import java.util.Optional;
import java.util.UUID;
import com.banking.dto.CreditResult;
import com.banking.dto.DebitResult;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.argThat;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
public class TransactionServiceTest 
{
    @Mock private TransactionRepository transactionRepository;
    @Mock private AccountRepository accountRepository;
    @InjectMocks private TransactionService transactionService;

    private UUID fromId;
    private UUID toId;
    private Account sender;
    private Account receiver;
    private Transaction mockTransaction;

    @BeforeEach
    void setUp() 
    {
        fromId = UUID.randomUUID();
        toId = UUID.randomUUID();

        sender = Account.builder().id(fromId).name("Likhith").balance(new BigDecimal("5000.00")).tpin("1234").build();

        receiver = Account.builder().id(toId).name("Arigela").balance(new BigDecimal("2000.00")).tpin("5678").build();

        mockTransaction = Transaction.builder().id(UUID.randomUUID()).fromAccount(fromId).toAccount(toId).amount(new BigDecimal("1000.00")).transferMode("NEFT").status("SUCCESS").build();
    }

    @Test
    void test_getTransaction_shouldReturnTransactionWhenValidIdProvided() 
    {
        UUID txnId = mockTransaction.getId();

        when(transactionRepository.findById(txnId)).thenReturn(Optional.of(mockTransaction));

        Optional<Transaction> result = transactionService.getTransaction(txnId);

        assertThat(result).isPresent();

        assertThat(result.get().getStatus()).isEqualTo("SUCCESS");
        assertThat(result.get().getAmount()).isEqualByComparingTo("1000.00");
        assertThat(result.get().getTransferMode()).isEqualTo("NEFT");

        verify(transactionRepository, times(1)).findById(txnId);
    }

    @Test
    void test_getTransaction_shouldReturnEmptyWhenTransactionNotFound() 
    {
        UUID unknownId = UUID.randomUUID();

        when(transactionRepository.findById(unknownId)).thenReturn(Optional.empty());

        Optional<Transaction> result = transactionService.getTransaction(unknownId);

        assertThat(result).isEmpty();

        verify(transactionRepository, times(1)).findById(unknownId);
    }

    @Test
    void test_getAllTransactions_shouldReturnListWhenTransactionsExist() 
    {
        when(transactionRepository.findAll()).thenReturn(List.of(mockTransaction));

        List<Transaction> result = transactionService.getAllTransactions();

        assertThat(result).hasSize(1);

        assertThat(result.get(0).getStatus()).isEqualTo("SUCCESS");

        verify(transactionRepository, times(1)).findAll();
    }

    @Test
    void test_getAllTransactions_shouldReturnEmptyListWhenNoTransactionsExist() 
    {
        when(transactionRepository.findAll()).thenReturn(Collections.emptyList());

        List<Transaction> result = transactionService.getAllTransactions();

        assertThat(result).isEmpty();

        verify(transactionRepository, times(1)).findAll();
    }

    @Test
    void test_getAccountTransactions_shouldReturnSuccessTransactionsWhenAccountIsReceiver() 
    {
        UUID thirdId = UUID.randomUUID();

        Transaction incomingSuccess = Transaction.builder().id(UUID.randomUUID()).fromAccount(thirdId).toAccount(fromId).amount(new BigDecimal("500.00")).transferMode("UPI").status("SUCCESS").build();

        when(transactionRepository.findByFromAccountOrToAccount(fromId, fromId)).thenReturn(List.of(incomingSuccess));

        List<Transaction> result = transactionService.getAccountTransactions(fromId);

        assertThat(result).hasSize(1);

        assertThat(result.get(0).getStatus()).isEqualTo("SUCCESS");
    }

    @Test
    void test_getAccountTransactions_shouldReturnAllSenderTransactionsRegardlessOfStatus() 
    {
        Transaction failedOutgoing = Transaction.builder().id(UUID.randomUUID()).fromAccount(fromId).toAccount(toId).amount(new BigDecimal("200.00")).transferMode("IMPS").status("FAILED").build();

        when(transactionRepository.findByFromAccountOrToAccount(fromId, fromId)).thenReturn(List.of(failedOutgoing));

        List<Transaction> result = transactionService.getAccountTransactions(fromId);

        assertThat(result).hasSize(1);

        assertThat(result.get(0).getStatus()).isEqualTo("FAILED");
    }

    @Test
    void test_getAccountTransactions_shouldExcludeFailedIncomingTransactions() 
    {
        UUID thirdId = UUID.randomUUID();

        Transaction failedIncoming = Transaction.builder().id(UUID.randomUUID()).fromAccount(thirdId).toAccount(fromId).amount(new BigDecimal("300.00")).transferMode("NEFT").status("FAILED").build();

        when(transactionRepository.findByFromAccountOrToAccount(fromId, fromId)).thenReturn(List.of(failedIncoming));

        List<Transaction> result = transactionService.getAccountTransactions(fromId);

        assertThat(result).isEmpty();
    }

    @Test
    void test_getAccountTransactions_shouldReturnEmptyListWhenNoTransactionsExist() 
    {
        when(transactionRepository.findByFromAccountOrToAccount(fromId, fromId)).thenReturn(Collections.emptyList());

        List<Transaction> result = transactionService.getAccountTransactions(fromId);

        assertThat(result).isEmpty();
    }

    @Test
    void test_getAccountTransactions_shouldReturnMixedTransactionsWhenBothSentAndReceived() 
    {
        UUID thirdId = UUID.randomUUID();

        Transaction outgoingFailed = Transaction.builder().id(UUID.randomUUID()).fromAccount(fromId).toAccount(toId).amount(new BigDecimal("100.00")).transferMode("UPI").status("FAILED").build();

        Transaction incomingSuccess = Transaction.builder().id(UUID.randomUUID()).fromAccount(thirdId).toAccount(fromId).amount(new BigDecimal("400.00")).transferMode("NEFT").status("SUCCESS").build();

        Transaction incomingFailed = Transaction.builder().id(UUID.randomUUID()).fromAccount(thirdId).toAccount(fromId).amount(new BigDecimal("200.00")).transferMode("IMPS").status("FAILED").build();

        when(transactionRepository.findByFromAccountOrToAccount(fromId, fromId)).thenReturn(List.of(outgoingFailed, incomingSuccess, incomingFailed));

        List<Transaction> result = transactionService.getAccountTransactions(fromId);

        assertThat(result).hasSize(2);

        assertThat(result).extracting(Transaction::getStatus).containsExactlyInAnyOrder("FAILED", "SUCCESS");
    }

    @Test
    void test_saveTransaction_shouldReturnSavedTransactionWhenValidDetailsProvided() 
    {
        when(transactionRepository.save(argThat(txn -> txn.getFromAccount().equals(fromId) && txn.getToAccount().equals(toId) && txn.getAmount().compareTo(new BigDecimal("1000.00")) == 0 && txn.getTransferMode().equals("NEFT") && txn.getStatus().equals("PENDING")))).thenReturn(mockTransaction);

        Transaction result = transactionService.saveTransaction(fromId, toId, new BigDecimal("1000.00"), "NEFT", "PENDING");

        assertThat(result).isNotNull();

        assertThat(result.getStatus()).isEqualTo("SUCCESS");

        verify(transactionRepository, times(1)).save(argThat(txn -> txn.getFromAccount().equals(fromId) && txn.getToAccount().equals(toId) && txn.getAmount().compareTo(new BigDecimal("1000.00")) == 0 && txn.getTransferMode().equals("NEFT") && txn.getStatus().equals("PENDING")));
    }

    @Test
    void test_saveTransaction_shouldPersistCorrectFieldsWhenCalled() 
    {
        when(transactionRepository.save(argThat(txn -> txn.getFromAccount().equals(fromId) && txn.getToAccount().equals(toId) && txn.getAmount().compareTo(new BigDecimal("1000.00")) == 0 && txn.getTransferMode().equals("RTGS") && txn.getStatus().equals("PENDING")))).thenReturn(mockTransaction);

        transactionService.saveTransaction(fromId, toId, new BigDecimal("1000.00"), "RTGS", "PENDING");

        verify(transactionRepository).save(argThat(txn -> txn.getFromAccount().equals(fromId) && txn.getToAccount().equals(toId) && txn.getAmount().compareTo(new BigDecimal("1000.00")) == 0 && txn.getTransferMode().equals("RTGS") && txn.getStatus().equals("PENDING")));
    }

    @Test
    void test_saveTransaction_shouldCallRepositorySaveOnce() 
    {
        when(transactionRepository.save(argThat(txn -> txn.getFromAccount().equals(fromId) && txn.getToAccount().equals(toId) && txn.getAmount().compareTo(new BigDecimal("1000.00")) == 0 && txn.getTransferMode().equals("NEFT") && txn.getStatus().equals("PENDING")))).thenReturn(mockTransaction);

        transactionService.saveTransaction(fromId, toId, new BigDecimal("1000.00"), "NEFT", "PENDING");

        verify(transactionRepository, times(1)).save(argThat(txn -> txn.getFromAccount().equals(fromId) && txn.getToAccount().equals(toId) && txn.getAmount().compareTo(new BigDecimal("1000.00")) == 0 && txn.getTransferMode().equals("NEFT") && txn.getStatus().equals("PENDING")));

        verifyNoMoreInteractions(transactionRepository);
    }
    @Test
    void test_debitAccount_shouldReturnDebitResultWhenValidRequest()
    {
        when(accountRepository.findById(fromId)).thenReturn(Optional.of(sender));

        DebitResult result = transactionService.debitAccount(fromId, new BigDecimal("1000.00"), "TXN-REF-001", "1234");

        assertThat(result.getAccountId()).isEqualTo(fromId);
        assertThat(result.getNewBalance()).isEqualByComparingTo("4000.00");
        assertThat(result.getTransferRef()).isEqualTo("TXN-REF-001");
    }

    @Test
    void test_debitAccount_shouldPersistUpdatedBalanceWhenValidRequest()
    {
        when(accountRepository.findById(fromId)).thenReturn(Optional.of(sender));

        transactionService.debitAccount(fromId, new BigDecimal("1000.00"), "TXN-REF-001", "1234");

        verify(accountRepository, times(1)).save(argThat(acc -> acc.getId().equals(fromId) && acc.getBalance().compareTo(new BigDecimal("4000.00")) == 0));
    }

    @Test
    void test_debitAccount_shouldAllowDebitWhenAmountEqualsBalance()
    {
        when(accountRepository.findById(fromId)).thenReturn(Optional.of(sender));

        DebitResult result = transactionService.debitAccount(fromId, new BigDecimal("5000.00"), "TXN-REF-002", "1234");

        assertThat(result.getNewBalance()).isEqualByComparingTo("0.00");
    }

    @Test
    void test_debitAccount_shouldThrowWhenAccountNotFound()
    {
        UUID unknownId = UUID.randomUUID();

        when(accountRepository.findById(unknownId)).thenReturn(Optional.empty());

        assertThatThrownBy(() -> transactionService.debitAccount(unknownId, new BigDecimal("100.00"), "TXN-REF-003", "1234"))
                .isInstanceOf(RuntimeException.class)
                .hasMessageContaining("Account not found");

        verify(accountRepository, never()).save(any());
    }

    @Test
    void test_debitAccount_shouldThrowWhenTpinIsIncorrect()
    {
        when(accountRepository.findById(fromId)).thenReturn(Optional.of(sender));

        assertThatThrownBy(() -> transactionService.debitAccount(fromId, new BigDecimal("100.00"), "TXN-REF-004", "0000"))
                .isInstanceOf(RuntimeException.class)
                .hasMessageContaining("invalid tpin");

        verify(accountRepository, never()).save(any());
    }

    @Test
    void test_debitAccount_shouldThrowWhenBalanceIsInsufficient()
    {
        when(accountRepository.findById(fromId)).thenReturn(Optional.of(sender));

        assertThatThrownBy(() -> transactionService.debitAccount(fromId, new BigDecimal("10000.00"), "TXN-REF-005", "1234"))
                .isInstanceOf(RuntimeException.class)
                .hasMessageContaining("insufficient balance");

        verify(accountRepository, never()).save(any());
    }

    // ---------- creditAccount ----------

    @Test
    void test_creditAccount_shouldReturnCreditResultWhenValidRequest()
    {
        when(accountRepository.findById(toId)).thenReturn(Optional.of(receiver));

        CreditResult result = transactionService.creditAccount(toId, new BigDecimal("500.00"), "TXN-REF-006");

        assertThat(result.getAccountId()).isEqualTo(toId);
        assertThat(result.getNewBalance()).isEqualByComparingTo("2500.00");
        assertThat(result.getTransferRef()).isEqualTo("TXN-REF-006");
    }

    @Test
    void test_creditAccount_shouldPersistUpdatedBalanceWhenValidRequest()
    {
        when(accountRepository.findById(toId)).thenReturn(Optional.of(receiver));

        transactionService.creditAccount(toId, new BigDecimal("500.00"), "TXN-REF-006");

        verify(accountRepository, times(1)).save(argThat(acc -> acc.getId().equals(toId) && acc.getBalance().compareTo(new BigDecimal("2500.00")) == 0));
    }

    @Test
    void test_creditAccount_shouldThrowWhenAccountNotFound()
    {
        UUID unknownId = UUID.randomUUID();

        when(accountRepository.findById(unknownId)).thenReturn(Optional.empty());

        assertThatThrownBy(() -> transactionService.creditAccount(unknownId, new BigDecimal("500.00"), "TXN-REF-007"))
                .isInstanceOf(RuntimeException.class)
                .hasMessageContaining("Account not found");

        verify(accountRepository, never()).save(any());
    }
}