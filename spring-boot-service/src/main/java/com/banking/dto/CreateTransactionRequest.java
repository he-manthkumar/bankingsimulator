package com.banking.dto;

import jakarta.validation.constraints.DecimalMin;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import lombok.Data;

import java.math.BigDecimal;
import java.util.UUID;

@Data
public class CreateTransactionRequest {
    @NotNull(message = "From account is required")
    private UUID fromAccount;

    @NotNull(message = "To account is required")
    private UUID toAccount;

    @NotNull(message = "Amount is required")
    @DecimalMin(value = "0.01", message = "Amount must be positive")
    private BigDecimal amount;

    @NotBlank(message = "Transfer mode is required")
    private String transferMode;

    private String status = "PENDING";
    
    private String tpin;
    
    private String correlationId;
}
