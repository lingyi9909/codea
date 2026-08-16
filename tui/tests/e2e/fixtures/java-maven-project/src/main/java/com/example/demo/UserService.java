package com.example.demo;

import com.example.demo.dto.UserDto;
import org.springframework.stereotype.Service;

import java.util.List;

@Service
public class UserService {

    public List<UserDto> listUsers() {
        return List.of();
    }
}
