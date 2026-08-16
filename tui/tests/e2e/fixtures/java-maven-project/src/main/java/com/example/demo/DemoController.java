package com.example.demo;

import com.example.demo.dto.CreateUserRequest;
import com.example.demo.dto.UserDto;
import com.example.demo.enums.UserStatus;
import jakarta.validation.Valid;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequestMapping("/api/users")
public class DemoController {

    @GetMapping("/{id}")
    public ResponseEntity<UserDto> getUser(@PathVariable Long id) {
        return ResponseEntity.ok(new UserDto());
    }

    @PostMapping
    public ResponseEntity<UserDto> createUser(@Valid @RequestBody CreateUserRequest request) {
        return ResponseEntity.ok(new UserDto());
    }

    @GetMapping
    public ResponseEntity<List<UserDto>> listUsers(
            @RequestParam @Min(1) Integer page,
            @RequestParam(required = false) UserStatus status) {
        return ResponseEntity.ok(List.of());
    }

    @DeleteMapping("/{id}")
    public ResponseEntity<Void> deleteUser(@PathVariable Long id) {
        return ResponseEntity.noContent().build();
    }

    @ExceptionHandler(IllegalArgumentException.class)
    @ResponseStatus(code = HttpStatus.BAD_REQUEST)
    public void handleBadRequest() {
    }
}
