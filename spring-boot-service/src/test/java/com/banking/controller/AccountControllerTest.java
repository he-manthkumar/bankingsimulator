package com.banking.controller;

import com.banking.model.Account;
import com.banking.model.Transaction;
import com.banking.service.AccountService;
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

@WebMvcTest(AccountController.class)
public class AccountControllerTest {

    @Autowired
    private MockMvc mockMvc;
    @MockBean
    private AccountService accountService;
    @MockBean
    private TransactionService transactionService;
    @Autowired
    private ObjectMapper objectMapper;
    private Account mockAccount;
    private UUID accountId;

    @BeforeEach
    void setUp() {
        accountId = UUID.randomUUID();
        mockAccount = new Account();
        mockAccount.setId(accountId);
        mockAccount.setName("Likhith");
        mockAccount.setBalance(new BigDecimal("5000.00"));
        mockAccount.setTpin("1234");
    }

    @Test
    void test_createAccount_shouldReturnAccountWhenAllFieldsProvided() throws Exception {
        Map<String, Object> body = new HashMap<>();
        body.put("name", "Likhith");
        body.put("initialBalance", "5000.00");
        body.put("tpin", "1234");

        when(accountService.createAccount("Likhith", new BigDecimal("5000.00"), "1234")).thenReturn(mockAccount);

        mockMvc.perform(post("/accounts").contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(body)))
                .andExpect(status().isOk()).andExpect(jsonPath("$.name").value("Likhith"))
                .andExpect(jsonPath("$.balance").value(5000.00));
    }

    @Test
    void test_createAccount_shouldReturnAccountWhenInitialBalanceIsZero() throws Exception {
        Map<String, Object> body = new HashMap<>();
        body.put("name", "Likhith");
        body.put("initialBalance", "0");
        body.put("tpin", "9999");

        Account zeroBalanceAccount = new Account();
        zeroBalanceAccount.setId(UUID.randomUUID());
        zeroBalanceAccount.setName("Likhith");
        zeroBalanceAccount.setBalance(BigDecimal.ZERO);
        zeroBalanceAccount.setTpin("9999");

        when(accountService.createAccount("Likhith", BigDecimal.ZERO, "9999")).thenReturn(zeroBalanceAccount);

        mockMvc.perform(post("/accounts").contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(body)))
                .andExpect(status().isOk()).andExpect(jsonPath("$.balance").value(0));
    }

    @Test
    void test_createAccount_shouldReturn400WhenNameNotProvided() throws Exception {
        Map<String, Object> body = new HashMap<>();
        body.put("initialBalance", "5000.00");
        body.put("tpin", "1234");
        mockMvc.perform(post("/accounts").contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(body)))
                .andExpect(status().isBadRequest());
    }

    @Test
    void test_createAccount_shouldReturn400WhenInitialBalanceNotProvided() throws Exception {
        Map<String, Object> body = new HashMap<>();
        body.put("name", "Likhith");
        body.put("tpin", "1234");
        mockMvc.perform(post("/accounts").contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(body)))
                .andExpect(status().isBadRequest());
    }
    @Test
    void test_createAccount_shouldReturn400WhenTpinNotProvided() throws Exception {
        Map<String, Object> body = new HashMap<>();
        body.put("name", "Likhith");
        body.put("initialBalance", "3000.00");

        mockMvc.perform(post("/accounts")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(body)))
                .andExpect(status().isBadRequest());
    }

    @Test
    void test_createAccount_shouldReturn400WhenAccountBalanceNegativeisGiven() throws Exception {
        Map<String, Object> body = new HashMap<>();
        body.put("name", "Likhith");
        body.put("initialBalance", "-100.00");
        body.put("tpin", "9999");

        mockMvc.perform(post("/accounts").contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(body)))
                .andExpect(status().isBadRequest());
    }


    @Test
    void test_getAllAccounts_shouldReturnListWhenAccountsExist() throws Exception {
        when(accountService.getAllAccounts()).thenReturn(List.of(mockAccount));

        mockMvc.perform(get("/accounts")).andExpect(status().isOk()).andExpect(jsonPath("$.length()").value(1))
                .andExpect(jsonPath("$[0].name").value("Likhith"))
                .andExpect(jsonPath("$[0].balance").value(5000.00))
                .andExpect(jsonPath("$[0].tpin").doesNotExist());
    }

    @Test
    void test_getAllAccounts_shouldReturnEmptyListWhenNoAccountsExist() throws Exception {
        when(accountService.getAllAccounts()).thenReturn(Collections.emptyList());
        mockMvc.perform(get("/accounts")).andExpect(status().isOk())
                .andExpect(jsonPath("$.length()").value(0));
    }

    @Test
    void test_getAccount_shouldReturnAccountWhenValidIdProvided() throws Exception {
        when(accountService.getAccount(accountId)).thenReturn(Optional.of(mockAccount));
        mockMvc.perform(get("/accounts/{id}", accountId)).andExpect(status().isOk())
                .andExpect(jsonPath("$.name").value("Likhith"))
                .andExpect(jsonPath("$.balance").value(5000.00))
                .andExpect(jsonPath("$.tpin").doesNotExist());
    }

    @Test
    void test_getAccount_shouldReturn404WhenAccountNotFound() throws Exception {
        UUID unknownId = UUID.randomUUID();
        when(accountService.getAccount(unknownId)).thenReturn(Optional.empty());
        mockMvc.perform(get("/accounts/{id}", unknownId))
                .andExpect(status().isNotFound());
    }

    @Test
    void test_getAccountTransactions_shouldReturnTransactionsWhenAccountHasTransactions() throws Exception {
        Transaction tx = new Transaction();
        tx.setId(UUID.randomUUID());
        tx.setAmount(new BigDecimal("200.00"));
        when(transactionService.getAccountTransactions(accountId)).thenReturn(List.of(tx));
        mockMvc.perform(get("/accounts/{id}/transactions", accountId)).andExpect(status().isOk())
                .andExpect(jsonPath("$.length()").value(1))
                .andExpect(jsonPath("$[0].amount").value(200.00));
    }

    @Test
    void test_getAccountTransactions_shouldReturnEmptyListWhenNoTransactionsExist() throws Exception {
        when(transactionService.getAccountTransactions(accountId)).thenReturn(Collections.emptyList());
        mockMvc.perform(get("/accounts/{id}/transactions", accountId)).andExpect(status().isOk())
                .andExpect(jsonPath("$.length()").value(0));
    }

    @Test
    void test_getAccountTransactions_shouldReturnEmptyListWhenAccountIdDoesNotExist() throws Exception {
        UUID unknownId = UUID.randomUUID();
        when(transactionService.getAccountTransactions(unknownId)).thenReturn(Collections.emptyList());
        mockMvc.perform(get("/accounts/{id}/transactions", unknownId)).andExpect(status().isOk())
                .andExpect(jsonPath("$.length()").value(0));
    }
}