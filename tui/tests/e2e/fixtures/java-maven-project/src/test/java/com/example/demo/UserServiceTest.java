package com.example.demo;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertTrue;

class UserServiceTest {

    @Test
    void listUsersReturnsEmptyList() {
        UserService service = new UserService();
        assertTrue(service.listUsers().isEmpty());
    }
}
