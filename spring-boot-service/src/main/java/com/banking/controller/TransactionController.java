package com.banking.controller;

import com.banking.model.Transaction;
import com.banking.service.TransactionService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import com.banking.dto.CreateTransactionRequest;
import jakarta.validation.Valid;
import java.util.List;
import java.util.UUID;

@RestController
@RequestMapping("/transactions")
public class TransactionController {

    private final TransactionService transactionService;

    public TransactionController(TransactionService transactionService) {
        this.transactionService = transactionService;
    }

    @PostMapping
    public ResponseEntity<Transaction> saveTransaction(@Valid @RequestBody CreateTransactionRequest body) {
        Transaction txn = transactionService.saveTransaction(
                body.getFromAccount(),
                body.getToAccount(),
                body.getAmount(),
                body.getTransferMode(),
                body.getStatus());
        return ResponseEntity.ok(txn);
    }

    @GetMapping("/{id}")
    public ResponseEntity<Transaction> getTransaction(@PathVariable UUID id) {
        return transactionService.getTransaction(id).map(ResponseEntity::ok).orElse(ResponseEntity.notFound().build());
    }

    @GetMapping("/all")
    public ResponseEntity<List<Transaction>> getAllTransactions() {
        return ResponseEntity.ok(transactionService.getAllTransactions());
    }
}