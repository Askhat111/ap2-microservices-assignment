# Order & Payment Platform - Clean Architecture Microservices

## Architecture Decisions & Bounded Contexts
This platform is decomposed into two distinct microservices: **Order Service** and **Payment Service**. 
Each service operates within its own **Bounded Context**, meaning they own their specific domain logic and data without sharing any models or databases. 
- **Order Context**: Manages the lifecycle of a customer's order (Pending, Paid, Failed, Cancelled).
- **Payment Context**: Manages financial transaction validations and enforces business rules (declining amounts > 100,000 cents).

The codebase follows **Clean Architecture** principles. The business logic inside the `usecase` layer is completely isolated from external frameworks. The `delivery` layer (Gin HTTP handlers) and `repository` layer (PostgreSQL) depend on domain interfaces, adhering to the Dependency Inversion Principle.

## Failure Handling
Inter-service communication is handled synchronously via REST. To ensure resilience and prevent cascading failures:
1. The Order Service communicates with the Payment Service using a dedicated Gateway with an explicit **2-second timeout** on the `http.Client`.
2. If the Payment Service is down, unresponsive, or returns an error, the Order Service catches this timeout.
3. It immediately updates the local Order state in the database to `Failed` and returns a `503 Service Unavailable` to the client. This guarantees state consistency instead of leaving orders permanently "Pending".

## API Examples (cURL)

**Create a Successful Order(Amount ≤ 100,000 cents):**
```bash
curl.exe -X POST http://localhost:8080/orders -H "Content-Type: application/json" -d "{\`"customer_id\`": \`"user_01\`", \`"item_name\`": \`"Keyboard\`", \`"amount\`": 45000}"
```
**Create a Declined Order (Amount > 100,000 cents):** 
```bash
curl.exe -X POST http://localhost:8080/orders -H "Content-Type: application/json" -d "{\`"customer_id\`": \`"user_01\`", \`"item_name\`": \`"Laptop\`", \`"amount\`": 150000}"
```
**Test Service Failure (503 Service Unavailable):**
```bash
curl.exe -X POST http://localhost:8080/orders -H "Content-Type: application/json" -d "{\`"customer_id\`": \`"std_003\`", \`"item_name\`": \`"Mouse\`", \`"amount\`": 2500}"
```
**Check Order Status:**
```bash
curl.exe -X GET http://localhost:8080/orders/{order_id}
```
**Cancel a Pending/Failed Order:**
```bash
curl.exe -X PATCH http://localhost:8080/orders/{order_id}/cancel
```

```mermaid
graph TD
    Client([Web Client / Postman]) -->|HTTP POST /orders| API_Gateway(Order Service :8080)
    
    subgraph Order Bounded Context
        API_Gateway --> UC_Order(Order Use Case)
        UC_Order --> Repo_Order(Order Repository)
        Repo_Order -->|Read/Write| DB_Order[(Order PostgreSQL :5432)]
    end

    subgraph Payment Bounded Context
        UC_Payment(Payment Use Case) --> Repo_Payment(Payment Repository)
        Repo_Payment -->|Read/Write| DB_Payment[(Payment PostgreSQL :5433)]
    end

    UC_Order -.->|REST HTTP POST /payments\nwith Timeout| UC_Payment