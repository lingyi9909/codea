package com.example.demo.dto;

import jakarta.validation.constraints.Email;
import jakarta.validation.constraints.NotBlank;

public class UserDto {

    private Long id;

    @NotBlank
    private String name;

    @Email
    private String email;

    private String status;
}
