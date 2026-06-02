package com.banking.controller;

import com.banking.model.Transaction;
import com.banking.service.TransactionService;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;

import java.math.BigDecimal;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;
import static org.mockito.Mockito.verify;

@WebMvcTest(TransactionController.class)
public class TransactionControllerTest {

        @Autowired
        private MockMvc mockMvc;
        @MockBean
        private TransactionService transactionService;
        @Autowired
        private ObjectMapper objectMapper;

        private Transaction mockTransaction;
        private UUID transactionId;
        private UUID fromAccountId;
        private UUID toAccountId;

        @BeforeEach
        void setUp() {
                transactionId = UUID.randomUUID();
                fromAccountId = UUID.randomUUID();
                toAccountId = UUID.randomUUID();
                mockTransaction = new Transaction();
                mockTransaction.setId(transactionId);
                mockTransaction.setFromAccount(fromAccountId);
                mockTransaction.setToAccount(toAccountId);
                mockTransaction.setAmount(new BigDecimal("1500.00"));
                mockTransaction.setTransferMode("NEFT");
                mockTransaction.setStatus("PENDING");
        }

        @Test
        void test_saveTransaction_shouldReturnTransactionWhenAllFieldsProvided() throws Exception {
                Map<String, Object> body = new HashMap<>();
                body.put("fromAccount", fromAccountId.toString());
                body.put("toAccount", toAccountId.toString());
                body.put("amount", "1500.00");
                body.put("transferMode", "NEFT");
                body.put("status", "PENDING");

                when(transactionService.saveTransaction(fromAccountId, toAccountId, new BigDecimal("1500.00"), "NEFT",
                                "PENDING"))
                                .thenReturn(mockTransaction);

                mockMvc.perform(post("/transactions").contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(body)))
                                .andExpect(status().isOk()).andExpect(jsonPath("$.amount").value(1500.00))
                                .andExpect(jsonPath("$.transferMode").value("NEFT"))
                                .andExpect(jsonPath("$.status").value("PENDING"));
                verify(transactionService).saveTransaction(fromAccountId, toAccountId, new BigDecimal("1500.00"),
                                "NEFT", "PENDING");
        }

        @Test
        void test_saveTransaction_shouldReturnTransactionWithDefaultStatusWhenStatusNotProvided() throws Exception {
                Map<String, Object> body = new HashMap<>();
                body.put("fromAccount", fromAccountId.toString());
                body.put("toAccount", toAccountId.toString());
                body.put("amount", "500.00");
                body.put("transferMode", "IMPS");

                Transaction defaultStatusTxn = new Transaction();
                defaultStatusTxn.setId(UUID.randomUUID());
                defaultStatusTxn.setStatus("PENDING");
                defaultStatusTxn.setAmount(new BigDecimal("500.00"));
                defaultStatusTxn.setTransferMode("IMPS");

                when(transactionService.saveTransaction(fromAccountId, toAccountId, new BigDecimal("500.00"), "IMPS",
                                "PENDING"))
                                .thenReturn(defaultStatusTxn);

                mockMvc.perform(post("/transactions").contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(body)))
                                .andExpect(status().isOk()).andExpect(jsonPath("$.status").value("PENDING"));
                verify(transactionService).saveTransaction(fromAccountId, toAccountId, new BigDecimal("500.00"), "IMPS",
                                "PENDING");
        }

        @Test
        void test_saveTransaction_shouldReturnTransactionWhenStatusIsSuccess() throws Exception {
                Map<String, Object> body = new HashMap<>();
                body.put("fromAccount", fromAccountId.toString());
                body.put("toAccount", toAccountId.toString());
                body.put("amount", "200.00");
                body.put("transferMode", "UPI");
                body.put("status", "SUCCESS");

                Transaction successTxn = new Transaction();
                successTxn.setId(UUID.randomUUID());
                successTxn.setStatus("SUCCESS");
                successTxn.setAmount(new BigDecimal("200.00"));
                successTxn.setTransferMode("UPI");

                when(transactionService.saveTransaction(fromAccountId, toAccountId, new BigDecimal("200.00"), "UPI",
                                "SUCCESS"))
                                .thenReturn(successTxn);

                mockMvc.perform(post("/transactions").contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(body)))
                                .andExpect(status().isOk()).andExpect(jsonPath("$.status").value("SUCCESS"));
                verify(transactionService).saveTransaction(fromAccountId, toAccountId, new BigDecimal("200.00"), "UPI",
                                "SUCCESS");
        }

        @Test
        void test_settleTransfer_shouldReturnBadRequestWhenTpinIsBlank() throws Exception {
                Map<String, Object> body = new HashMap<>();
                body.put("fromAccount", fromAccountId.toString());
                body.put("toAccount", toAccountId.toString());
                body.put("amount", "300.00");
                body.put("transferMode", "UPI");
                body.put("tpin", "");

                mockMvc.perform(post("/transactions/settle")
                                .contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(body)))
                                .andExpect(status().isBadRequest());
        }

        @Test
        void test_settleTransfer_shouldReturnTransactionWhenAllFieldsProvided() throws Exception {
                Map<String, Object> body = new HashMap<>();
                body.put("fromAccount", fromAccountId.toString());
                body.put("toAccount", toAccountId.toString());
                body.put("amount", "1500.00");
                body.put("transferMode", "NEFT");
                body.put("status", "PENDING");
                body.put("tpin", "1234");

                Transaction settledTxn = new Transaction();
                settledTxn.setId(UUID.randomUUID());
                settledTxn.setStatus("SUCCESS");
                settledTxn.setAmount(new BigDecimal("1500.00"));
                settledTxn.setTransferMode("NEFT");

                when(transactionService.settleTransfer(fromAccountId, toAccountId, new BigDecimal("1500.00"), "NEFT",
                                "1234"))
                                .thenReturn(settledTxn);

                mockMvc.perform(post("/transactions/settle").contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(body)))
                                .andExpect(status().isOk()).andExpect(jsonPath("$.status").value("SUCCESS"));
                verify(transactionService).settleTransfer(fromAccountId, toAccountId, new BigDecimal("1500.00"), "NEFT",
                                "1234");
        }

        @Test
        void test_settleTransfer_shouldReturnSuccessWhenOnlyRequiredFieldsProvided() throws Exception {
                Map<String, Object> body = new HashMap<>();
                body.put("fromAccount", fromAccountId.toString());
                body.put("toAccount", toAccountId.toString());
                body.put("amount", "800.00");
                body.put("transferMode", "RTGS");
                body.put("tpin", "5678");

                Transaction settledTxn = new Transaction();
                settledTxn.setId(UUID.randomUUID());
                settledTxn.setStatus("SUCCESS");
                settledTxn.setAmount(new BigDecimal("800.00"));

                when(transactionService.settleTransfer(fromAccountId, toAccountId, new BigDecimal("800.00"), "RTGS",
                                "5678"))
                                .thenReturn(settledTxn);

                mockMvc.perform(post("/transactions/settle").contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(body)))
                                .andExpect(status().isOk()).andExpect(jsonPath("$.status").value("SUCCESS"));
                verify(transactionService).settleTransfer(fromAccountId, toAccountId, new BigDecimal("800.00"), "RTGS",
                                "5678");
        }

        @Test
        void test_settleTransfer_shouldReturnTransactionWithEmptyTpinWhenTpinNotProvided() throws Exception {
                Map<String, Object> body = new HashMap<>();
                body.put("fromAccount", fromAccountId.toString());
                body.put("toAccount", toAccountId.toString());
                body.put("amount", "300.00");
                body.put("transferMode", "UPI");

                mockMvc.perform(post("/transactions/settle").contentType(MediaType.APPLICATION_JSON)
                                .content(objectMapper.writeValueAsString(body)))
                                .andExpect(status().isBadRequest());
        }

        @Test
        void test_getTransaction_shouldReturnTransactionWhenValidIdProvided() throws Exception {
                when(transactionService.getTransaction(transactionId)).thenReturn(Optional.of(mockTransaction));
                mockMvc.perform(get("/transactions/{id}", transactionId)).andExpect(status().isOk())
                                .andExpect(jsonPath("$.amount").value(1500.00))
                                .andExpect(jsonPath("$.transferMode").value("NEFT"));
                verify(transactionService).getTransaction(transactionId);
        }

        @Test
        void test_getTransaction_shouldReturn404WhenTransactionNotFound() throws Exception {
                UUID unknownId = UUID.randomUUID();
                when(transactionService.getTransaction(unknownId)).thenReturn(Optional.empty());
                mockMvc.perform(get("/transactions/{id}", unknownId)).andExpect(status().isNotFound());
                verify(transactionService).getTransaction(unknownId);
        }

        @Test
        void test_getAllTransactions_shouldReturnListWhenTransactionsExist() throws Exception {
                when(transactionService.getAllTransactions()).thenReturn(List.of(mockTransaction));
                mockMvc.perform(get("/transactions/all")).andExpect(status().isOk())
                                .andExpect(jsonPath("$.length()").value(1))
                                .andExpect(jsonPath("$[0].transferMode").value("NEFT"));
                verify(transactionService).getAllTransactions();
        }

        @Test
        void test_getAllTransactions_shouldReturnEmptyListWhenNoTransactionsExist() throws Exception {
                when(transactionService.getAllTransactions()).thenReturn(Collections.emptyList());
                mockMvc.perform(get("/transactions/all")).andExpect(status().isOk())
                                .andExpect(jsonPath("$.length()").value(0));
                verify(transactionService).getAllTransactions();
        }
}