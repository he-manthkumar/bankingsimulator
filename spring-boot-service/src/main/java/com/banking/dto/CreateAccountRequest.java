package com.banking.dto;

import jakarta.validation.constraints.DecimalMin;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import lombok.Data;
import jakarta.validation.constraints.Size;

import java.math.BigDecimal;

@Data
public class CreateAccountRequest {
    @NotBlank(message = "Name is required")
    private String name;

    @NotNull(message = "Initial balance is required")
    @DecimalMin(value = "0.0", message = "Balance must be non-negative")
    private BigDecimal initialBalance;

    @NotBlank(message = "TPIN is required")
    @Size(min = 4, max = 6, message = "TPIN must be between 4 and 6 characters")
    private String tpin;
}
