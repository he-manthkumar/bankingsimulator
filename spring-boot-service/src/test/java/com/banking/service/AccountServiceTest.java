package com.banking.service;

import com.banking.model.Account;
import com.banking.repository.AccountRepository;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.math.BigDecimal;
import java.util.Collections;
import java.util.List;
import java.util.Optional;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.argThat;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
public class AccountServiceTest {
    @Mock
    private AccountRepository accountRepository;
    @InjectMocks
    private AccountService accountService;

    private Account mockAccount;
    private UUID accountId;

    @BeforeEach
    void setUp() {
        accountId = UUID.randomUUID();

        mockAccount = Account.builder()
                .id(accountId)
                .name("Likhith")
                .balance(new BigDecimal("5000.00"))
                .tpin("1234")
                .build();
    }

    @Test
    void test_createAccount_shouldReturnSavedAccountWhenValidDetailsProvided() {
        when(accountRepository.save(argThat(account -> account.getName().equals("Likhith")
                && account.getBalance().compareTo(new BigDecimal("5000.00")) == 0 && account.getTpin().equals("1234"))))
                .thenReturn(mockAccount);

        Account result = accountService.createAccount("Likhith", new BigDecimal("5000.00"), "1234");

        assertThat(result).isNotNull();
        assertThat(result.getName()).isEqualTo("Likhith");
        assertThat(result.getBalance()).isEqualByComparingTo("5000.00");
        assertThat(result.getTpin()).isEqualTo("1234");

        verify(accountRepository, times(1)).save(argThat(account -> account.getName().equals("Likhith")
                && account.getBalance().compareTo(new BigDecimal("5000.00")) == 0 && account.getTpin().equals("1234")));
    }

    @Test
    void test_createAccount_shouldReturnSavedAccountWhenZeroBalanceProvided() {
        Account zeroBalanceAccount = Account.builder().id(UUID.randomUUID()).name("Likhith").balance(BigDecimal.ZERO)
                .tpin("0000").build();
        when(accountRepository.save(argThat(account -> account.getName().equals("Likhith")
                && account.getBalance().compareTo(BigDecimal.ZERO) == 0 && account.getTpin().equals("0000"))))
                .thenReturn(zeroBalanceAccount);

        Account result = accountService.createAccount("Likhith", BigDecimal.ZERO, "0000");

        assertThat(result.getBalance()).isEqualByComparingTo(BigDecimal.ZERO);

        verify(accountRepository, times(1)).save(argThat(account -> account.getName().equals("Likhith")
                && account.getBalance().compareTo(BigDecimal.ZERO) == 0 && account.getTpin().equals("0000")));
    }

    @Test
    void test_createAccount_shouldReturnSavedAccountWhenDefaultTpinProvided() {
        Account defaultTpinAccount = Account.builder().id(UUID.randomUUID()).name("Likhith")
                .balance(new BigDecimal("1000.00")).tpin("0000").build();

        when(accountRepository.save(argThat(account -> account.getName().equals("Likhith")
                && account.getBalance().compareTo(new BigDecimal("1000.00")) == 0 && account.getTpin().equals("0000"))))
                .thenReturn(defaultTpinAccount);

        Account result = accountService.createAccount("Likhith", new BigDecimal("1000.00"), "0000");

        assertThat(result.getTpin()).isEqualTo("0000");

        verify(accountRepository, times(1)).save(argThat(account -> account.getName().equals("Likhith")
                && account.getBalance().compareTo(new BigDecimal("1000.00")) == 0 && account.getTpin().equals("0000")));
    }

    @Test
    void test_createAccount_shouldCallRepositorySaveOnce() {
        when(accountRepository.save(argThat(account -> account.getName().equals("Likhith")
                && account.getBalance().compareTo(new BigDecimal("5000.00")) == 0 && account.getTpin().equals("1234"))))
                .thenReturn(mockAccount);

        accountService.createAccount("Likhith", new BigDecimal("5000.00"), "1234");

        verify(accountRepository, times(1)).save(argThat(account -> account.getName().equals("Likhith")
                && account.getBalance().compareTo(new BigDecimal("5000.00")) == 0 && account.getTpin().equals("1234")));

        verifyNoMoreInteractions(accountRepository);
    }

    @Test
    void test_getAccount_shouldReturnAccountWhenValidIdProvided() {
        when(accountRepository.findById(accountId)).thenReturn(Optional.of(mockAccount));

        Optional<Account> result = accountService.getAccount(accountId);

        assertThat(result).isPresent();
        assertThat(result.get().getId()).isEqualTo(accountId);
        assertThat(result.get().getName()).isEqualTo("Likhith");

        verify(accountRepository, times(1)).findById(accountId);
    }

    @Test
    void test_getAccount_shouldReturnEmptyWhenAccountNotFound() {
        UUID unknownId = UUID.randomUUID();

        when(accountRepository.findById(unknownId)).thenReturn(Optional.empty());

        Optional<Account> result = accountService.getAccount(unknownId);

        assertThat(result).isEmpty();

        verify(accountRepository, times(1)).findById(unknownId);
    }

    @Test
    void test_getAccount_shouldCallRepositoryFindByIdOnce() {
        when(accountRepository.findById(accountId)).thenReturn(Optional.of(mockAccount));

        accountService.getAccount(accountId);

        verify(accountRepository, times(1)).findById(accountId);

        verifyNoMoreInteractions(accountRepository);
    }

    @Test
    void test_getAllAccounts_shouldReturnListWhenAccountsExist() {
        when(accountRepository.findAll()).thenReturn(List.of(mockAccount));

        List<Account> result = accountService.getAllAccounts();

        assertThat(result).isNotEmpty();
        assertThat(result).hasSize(1);
        assertThat(result.get(0).getName()).isEqualTo("Likhith");

        verify(accountRepository, times(1)).findAll();
        verifyNoMoreInteractions(accountRepository);
    }

    @Test
    void test_getAllAccounts_shouldReturnEmptyListWhenNoAccountsExist() {
        when(accountRepository.findAll()).thenReturn(Collections.emptyList());

        List<Account> result = accountService.getAllAccounts();

        assertThat(result).isEmpty();

        verify(accountRepository, times(1)).findAll();
    }

    @Test
    void test_getAllAccounts_shouldReturnMultipleAccountsWhenMultipleAccountsExist() {
        Account secondAccount = Account.builder().id(UUID.randomUUID()).name("Arigela")
                .balance(new BigDecimal("2500.00")).tpin("5678").build();

        when(accountRepository.findAll()).thenReturn(List.of(mockAccount, secondAccount));

        List<Account> result = accountService.getAllAccounts();

        assertThat(result).hasSize(2);
        assertThat(result).extracting(Account::getName).containsExactly("Likhith", "Arigela");

        verify(accountRepository, times(1)).findAll();
    }

    @Test
    void test_getAllAccounts_shouldCallRepositoryFindAllOnce() {
        when(accountRepository.findAll()).thenReturn(Collections.emptyList());

        accountService.getAllAccounts();

        verify(accountRepository, times(1)).findAll();

        verifyNoMoreInteractions(accountRepository);
    }
}