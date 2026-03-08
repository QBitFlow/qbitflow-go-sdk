# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-10-23

### Added

-   Initial release of QBitFlow Go SDK
-   Complete API coverage for QBitFlow Cryptocurrency Payment API
-   One-time payment support
    -   Create payment sessions
    -   Retrieve payment details
    -   Get payment history with pagination
    -   Combined payment history (one-time + subscription)
-   Subscription management
    -   Create subscription sessions
    -   Support for trial periods
    -   Retrieve subscription details
    -   Get payment history
    -   Force cancel subscriptions
    -   Execute test billing cycles (test mode)
-   Pay-as-you-go subscription support
    -   Create PAYG sessions
    -   Free credits support
    -   Minimum periods configuration
    -   Increase usage units
    -   Usage tracking
-   Transaction status tracking
    -   Get transaction status
    -   Support for all transaction types
    -   Real-time status updates
-   Custom error types
    -   QBitFlowError for API errors
    -   NotFoundError for 404 responses
    -   ValidationError for client-side validation
-   Comprehensive documentation
    -   Detailed README with examples
    -   API reference documentation
    -   Code comments and doc strings
-   Example code
    -   Basic payment example
    -   Subscription management example
    -   PAYG subscription example
    -   Webhook handler example
    -   Complete payment flow example
-   Integration tests
    -   Payment session tests
    -   Subscription tests
    -   PAYG tests
    -   Status checking tests
    -   Error handling tests
-   Type-safe models
    -   All request/response models
    -   Duration support
    -   Currency models
    -   Transaction types and statuses
-   Configuration options
    -   Custom base URL
    -   Custom timeout
    -   API key authentication

### Developer Experience

-   Clean, idiomatic Go code
-   Comprehensive examples
-   Well-documented API
-   Easy integration
-   Production-ready error handling

[1.0.0]: https://github.com/qbitflow/qbitflow-go-sdk/releases/tag/v1.0.0


## [1.1.0] - 2026-03-08

### Added

-   HMAC signature verification for webhook requests
-   New `Verify` function in `WebhookService` to verify webhook authenticity
-   Updated documentation with webhook verification examples

### Security

-   Improved HMAC signature verification process
-   Enhanced input validation for webhook requests
