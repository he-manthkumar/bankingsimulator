package com.banking.service;

import com.banking.model.Account;
import com.banking.repository.AccountRepository;
import org.springframework.stereotype.Service;

import java.math.BigDecimal;
import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Service
public class AccountService 
{

    private final AccountRepository accountRepository;

    public AccountService(AccountRepository accountRepository) 
    {
        this.accountRepository = accountRepository;
    }

    public Account createAccount(String name, BigDecimal initialBalance, String tpin) 
    {
        Account account = Account.builder()
                .name(name)
                .balance(initialBalance)
                .tpin(tpin)
                .build();
        return accountRepository.save(account);
    }

    public Optional<Account> getAccount(UUID id) 
    {
        return accountRepository.findById(id);
    }

    public List<Account> getAllAccounts()
    {
        return accountRepository.findAll();
    }
}