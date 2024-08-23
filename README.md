# Project Description

This project facilitates simple User and Service Management with PostgreSQL for state management. 
Security is enforced by JWT based Authentication. 
Gin's middleware is leveraged to do Authorization based on User Roles embedded in Token.

As of now, refresh token is not introduced. If underlying application demands, 
security risks like Refresh Token Theft, Token Replay Attack, Extended Exposure parameters can be considered and implemented.

Project is backed by Go's recommended design pattern(s), with data-oriented as well as test driven approach.

Customised Foundation packages: [Logger](https://github.com/sabariarunkumar/go-logger), [PostgresInit](https://github.com/sabariarunkumar/go-postgresql-init)
## Directory Structure
```
├── cmd                      # entry-point
│   ├── api                  # init api-server       
│   └── migration            # migrate and exit
├── internal                 # bussiness logic modules 
│   ├── auth                 # jwt authn 
│   ├── components           # services
│   │   ├── role             # role management
│   │   ├── service          # service management
│   │   └── user             # user management
│   ├── configs              # app runtime config initializer
│   ├── errors               # defined runtime errors
│   ├── middleware           # intercepts request and facilitates authn/authz
│   ├── misc                 # misc
│   ├── models               # database models and related interfaces
│   └── utils                # utils   
├── tests                    # tests with explained scenarios
│   └── integration          # integration tests
└── vendor                   # pkg vendor 
```
## Building the application

```
make build
```

## Running Tests
#### Unit Test
Tests are defined for each module alongside with business files.
```
make unit-test
```
#### Integration Test

PreRequisite: PostgresqlDB, Runtime params in [runtime.env](https://github.com/sabariarunkumar/user-and-service-mgmt-go/blob/main/tests/integration/runtime.env) 

TestScenario available in [scenario.txt](https://github.com/sabariarunkumar/user-and-service-mgmt-go/blob/main/tests/integration/scenario.txt)
```
make integration-test
```

## Running the application

```
./bin/userservice
```

## Setup Instructions
### 1. Configure Runtime Parameters
-  Application accepts config params which can be set in a file or made available through environment variables.
  
  
   ```
   ./bin/userservice -env-file filename
   ```
- Runtime Parameters and their default values are shown below in expected env file format.
  ```
  # Server Configuration
   PORT=8080

   # Database Configuration
   DB_HOST=127.0.0.1
   DB_PORT=5432
   DB_USER=postgres
   DB_NAME=userservice
   DB_PASSWORD=postgres

   # Database Connection Pooling
   DB_MAX_CONN_IDLE_TIME_SEC=1800
   DB_MAX_OPEN_CONN=20

   # Logging
   LOG_LEVEL=info

   # JWT Configuration
   JWT_SECRET=userservice123
   JWT_EXPIRATION_IN_SECONDS=300
   ```
### 2. Run DB Migration
- Necessary tables and views will be migrated in this process
   ```
   ./bin/userservice --migrate true
   ```
- Upon running the migrations, a default admin user will be created automatically with temporary password
   - The default admin user will have the following credentials:
      ```
     Email: admin@mgmtportal.com
     Password: admin123
      ```
## User Management
1. Admin user is expected to do login through UI and frontend will check for a flag "password_change_required"
2. The flag is expected to be set , now UI is expected to redirect to change password page where user will update his password
3. Application comes up with default Roles
```
basic:    View services and their versions.
advanced: Fully Manage services; View users in systems
admin:    Fully Manage services/users in systems
```
4. Admin user(s) can add user into system with their name, email, role. Upon successful addition, a temporary password will be displayed to admin. This can be extended in future to send this temporary password to newly added user through e-mail.
5. Newly added user can login with this temporary password, eventually getting redirected to reset password page.

## Service Management
1. Authorized users can add services with metadata info like Service Name, description.
2. Versions can also be Configured as a part of service. Associated metadata info for versions are tag, info
3. Added services can be filtered, sorted by name and data [either ascending or descending], and paginated
4. Service versions can be filtered, sorted by data [only descending], and paginated.

## Performance Considerations
1. Rearranging struct fields based on their sizes in descending order can impact the memory layout and alignment, potentially leading to better cache utilization and reduced memory usage. Here we are trading it off with code readability.
2. Assuming high scalable users and small set of services, to support high performant reads, sorting over extendible columns [name, date] for every request wont be a good option, hence we are creating a materialized view in DB per column with services pre-sorted respectively. This strategy will have edge over DB's optimization techniques like indexing etc.
3. Materialized view sync triggers will be handled by application, such that in busy environment, we send the sync/refresh trigger in controlled manner
4. Since User base might be higher, and homepage traffic will proportionately increase, we pre-calculate total version count, and keep it tagged to service in DB, rather than calculating for every homepage visit.
