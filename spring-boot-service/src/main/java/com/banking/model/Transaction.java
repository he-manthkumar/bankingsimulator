package com.banking.model;

import jakarta.persistence.*;
import lombok.*;
import org.hibernate.annotations.CreationTimestamp;

import java.math.BigDecimal;
import java.time.LocalDateTime;
import java.util.UUID;

@Entity
@Table(name = "transactions")

@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
@Builder

public class Transaction 
{
    @Id @GeneratedValue(strategy = GenerationType.UUID)private UUID id;
    @Column(name = "correlation_id", unique = true) private UUID correlationId;
    @Column(name = "from_account", nullable = false) private UUID fromAccount;
    @Column(name = "to_account", nullable = false)private UUID toAccount;
    @Column(nullable = false, precision = 15, scale = 2)private BigDecimal amount;
    @Column(name = "transfer_mode", nullable = false, length = 20)private String transferMode;
    @Column(nullable = false, length = 20)private String status;
    @CreationTimestamp @Column(name = "created_at", updatable = false) private LocalDateTime createdAt;
}