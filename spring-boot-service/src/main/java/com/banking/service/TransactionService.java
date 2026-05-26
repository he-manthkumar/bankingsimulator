package com.banking.service;

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

    @Transactional
    public Transaction settleTransfer(UUID fromId, UUID toId, BigDecimal amount, String transferMode, String tpin) {
        Account sender = accountRepository.findById(fromId)
                .orElseThrow(() -> new RuntimeException("Sender account not found"));
        Account receiver = accountRepository.findById(toId)
                .orElseThrow(() -> new RuntimeException("Receiver account not found"));

        if (!sender.getTpin().equals(tpin)) {
            return transactionRepository.save(buildTransaction(fromId, toId, amount, transferMode, "FAILED"));
        }

        if (sender.getBalance().compareTo(amount) < 0) {
            return transactionRepository.save(buildTransaction(fromId, toId, amount, transferMode, "FAILED"));
        }

        sender.setBalance(sender.getBalance().subtract(amount));
        receiver.setBalance(receiver.getBalance().add(amount));
        accountRepository.save(sender);
        accountRepository.save(receiver);

        return transactionRepository.save(buildTransaction(fromId, toId, amount, transferMode, "SUCCESS"));
    }

    private Transaction buildTransaction(UUID fromId, UUID toId, BigDecimal amount, String transferMode,
            String status) {
        return Transaction.builder()
                .fromAccount(fromId)
                .toAccount(toId)
                .amount(amount)
                .transferMode(transferMode)
                .status(status)
                .build();
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
}