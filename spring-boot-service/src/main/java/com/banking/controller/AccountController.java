package com.banking.controller;

import com.banking.dto.CreditRequest;
import com.banking.dto.CreditResult;
import com.banking.dto.DebitRequest;
import com.banking.dto.DebitResult;
import com.banking.model.Account;
import com.banking.model.Transaction;
import com.banking.service.AccountService;
import com.banking.service.TransactionService;

import jakarta.validation.Valid;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import com.banking.dto.CreateAccountRequest;

import java.util.List;
import java.util.Map;
import java.util.UUID;

@RestController
@RequestMapping("/accounts")
public class AccountController {
    private final AccountService accountService;
    private final TransactionService transactionService;

    public AccountController(AccountService accountService, TransactionService transactionService) {
        this.accountService = accountService;
        this.transactionService = transactionService;
    }

    @PostMapping
    public ResponseEntity<Account> createAccount(@Valid @RequestBody CreateAccountRequest body) {
        Account account = accountService.createAccount(
                body.getName(),
                body.getInitialBalance(),
                body.getTpin());
        return ResponseEntity.ok(account);
    }

    @GetMapping
    public ResponseEntity<List<Account>> getAllAccounts() {
        return ResponseEntity.ok(accountService.getAllAccounts());
    }

    @GetMapping("/{id}")
    public ResponseEntity<Account> getAccount(@PathVariable UUID id) {
        return accountService.getAccount(id).map(ResponseEntity::ok).orElse(ResponseEntity.notFound().build());
    }

    @GetMapping("/{id}/transactions")
    public ResponseEntity<List<Transaction>> getAccountTransactions(@PathVariable UUID id) {
        return ResponseEntity.ok(transactionService.getAccountTransactions(id));
    }

    @PostMapping("/{id}/debit")
    public ResponseEntity<?> debitAccount(@PathVariable UUID id, @Valid @RequestBody DebitRequest body) {
        try {
            DebitResult result = transactionService.debitAccount(
                    id,
                    body.getAmount(),
                    body.getTransferRef(),
                    body.getTpin());
            return ResponseEntity.ok(result);
        } catch (RuntimeException e) {
            return ResponseEntity.badRequest().body(Map.of("error", e.getMessage()));
        }
    }

    @PostMapping("/{id}/credit")
    public ResponseEntity<?> creditAccount(@PathVariable UUID id, @Valid @RequestBody CreditRequest body) {
        try {
            CreditResult result = transactionService.creditAccount(
                    id,
                    body.getAmount(),
                    body.getTransferRef());
            return ResponseEntity.ok(result);
        } catch (RuntimeException e) {
            return ResponseEntity.badRequest().body(Map.of("error", e.getMessage()));
        }
    }
}