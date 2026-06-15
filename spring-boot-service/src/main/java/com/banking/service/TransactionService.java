package com.banking.service;

import com.banking.dto.CreditResult;
import com.banking.dto.DebitResult;
import com.banking.model.Account;
import com.banking.model.Transaction;
import com.banking.repository.AccountRepository;
import com.banking.repository.TransactionRepository;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.math.BigDecimal;
import java.util.List;
import java.util.Optional;
import java.util.UUID;
import java.util.stream.Collectors;

@Service
public class TransactionService {

    private final TransactionRepository transactionRepository;
    private final AccountRepository accountRepository;

    public TransactionService(TransactionRepository transactionRepository, AccountRepository accountRepository) {
        this.transactionRepository = transactionRepository;
        this.accountRepository = accountRepository;
    }


    public Optional<Transaction> getTransaction(UUID id) {
        return transactionRepository.findById(id);
    }

    public List<Transaction> getAllTransactions() {
        return transactionRepository.findAll();
    }

    public List<Transaction> getAccountTransactions(UUID accountId) {

        List<Transaction> all = transactionRepository.findByFromAccountOrToAccount(accountId, accountId);

        return all.stream().filter(txn -> txn.getStatus().equals("SUCCESS") || txn.getFromAccount().equals(accountId))
                .collect(Collectors.toList());
    }

    public Transaction saveTransaction(UUID fromId, UUID toId, BigDecimal amount, String transferMode, String status) {
        Transaction txn = Transaction.builder()
                .fromAccount(fromId)
                .toAccount(toId)
                .amount(amount)
                .transferMode(transferMode)
                .status(status)
                .build();
        return transactionRepository.save(txn);
    }

    @Transactional
    public DebitResult debitAccount(UUID accountId, BigDecimal amount, String transferRef, String tpin) {
        Account sender = accountRepository.findById(accountId)
                .orElseThrow(() -> new RuntimeException("Account not found: " + accountId));
        if (!sender.getTpin().equals(tpin)) {
            throw new RuntimeException("invalid tpin for account " + accountId);
        }
        if (sender.getBalance().compareTo(amount) < 0) {
            throw new RuntimeException("insufficient balance");
        }
        sender.setBalance(sender.getBalance().subtract(amount));
        accountRepository.save(sender);
        return DebitResult.builder()
                .accountId(accountId)
                .newBalance(sender.getBalance())
                .transferRef(transferRef)
                .build();
    }

    @Transactional
    public CreditResult creditAccount(UUID accountId, BigDecimal amount, String transferRef) {
        Account receiver = accountRepository.findById(accountId)
                .orElseThrow(() -> new RuntimeException("Account not found: " + accountId));
        receiver.setBalance(receiver.getBalance().add(amount));
        accountRepository.save(receiver);
        return CreditResult.builder()
                .accountId(accountId)
                .newBalance(receiver.getBalance())
                .transferRef(transferRef)
                .build();
    }
}