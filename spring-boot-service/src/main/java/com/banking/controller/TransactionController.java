package com.banking.controller;

import com.banking.model.Transaction;
import com.banking.service.TransactionService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import com.banking.dto.CreateTransactionRequest;
import jakarta.validation.Valid;
import java.math.BigDecimal;
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
        UUID fromAccount = body.getFromAccount();
        UUID toAccount = body.getToAccount();
        BigDecimal amount = body.getAmount();
        String transferMode = body.getTransferMode();
        String status = body.getStatus();
        Transaction txn = transactionService.saveTransaction(fromAccount, toAccount, amount, transferMode, status);
        return ResponseEntity.ok(txn);
    }

    @PostMapping("/settle")
    public ResponseEntity<Transaction> settleTransfer(@Valid @RequestBody CreateTransactionRequest body) {
        if (body.getTpin() == null || body.getTpin().isBlank()) {
            return ResponseEntity.badRequest().build();
        }
        UUID fromAccount = body.getFromAccount();
        UUID toAccount = body.getToAccount();
        BigDecimal amount = body.getAmount();
        String transferMode = body.getTransferMode();
        String tpin = body.getTpin();
        Transaction txn = transactionService.settleTransfer(fromAccount, toAccount, amount, transferMode, tpin);
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