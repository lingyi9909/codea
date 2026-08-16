package com.example.demo.dto;

public class UserDto {

    private Long id;

    @NotBlank
    private String name;

    @Email
    private String email;

    private String status;
}
