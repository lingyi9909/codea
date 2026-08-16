package com.example.demo.dto;

public class CreateUserRequest {

    @NotBlank
    private String name;

    @Email
    private String email;

    @Min(1)
    @Max(120)
    private Integer age;
}
