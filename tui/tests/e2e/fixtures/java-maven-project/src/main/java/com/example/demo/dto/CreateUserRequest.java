package com.example.demo.dto;

import jakarta.validation.constraints.Email;
import jakarta.validation.constraints.Max;
import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.NotBlank;

public class CreateUserRequest {

    @NotBlank
    private String name;

    @Email
    private String email;

    @Min(1)
    @Max(120)
    private Integer age;
}
