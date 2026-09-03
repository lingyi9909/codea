@echo off
rem Fixture mvnw.cmd stub. Emits the same deterministic Surefire-style summary
rem as mvnw so native Windows regression stays offline and dependency-free.
echo [INFO] Scanning for projects...
echo [INFO] ------------------------------------------------------------------------
echo [INFO] Building demo 1.0.0
echo [INFO] ------------------------------------------------------------------------
echo [INFO] --- maven-surefire-plugin:3.2.2:test (default-test) @ demo ---
echo [INFO] Running com.example.demo.UserServiceTest
echo [INFO] Tests run: 3, Failures: 0, Errors: 0, Skipped: 0, Time elapsed: 0.101 s
echo [INFO]
echo [INFO] Results:
echo [INFO]
echo [INFO] Tests run: 3, Failures: 0, Errors: 0, Skipped: 0
echo [INFO]
echo [INFO] ------------------------------------------------------------------------
echo [INFO] BUILD SUCCESS
echo [INFO] ------------------------------------------------------------------------
exit /b 0
