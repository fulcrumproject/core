## Description

This PR enhances the README.md file with comprehensive API usage examples and SDK code samples to help developers integrate with the Fulcrum Core API.

## Changes

- **API Usage Examples**: Added detailed curl examples for all major API endpoints including:
  - Authentication (token-based and OAuth)
  - Participant management (create, list, get)
  - Agent management (register, health checks)
  - Service management (create, update, list with filters)
  - Job queue operations (create, poll, update status)
  - Metrics collection and querying
  - Event subscriptions

- **SDK Client Libraries**: Added example implementations in:
  - **Go**: Complete client library example with participant creation
  - **Python**: Client class with participant and service listing examples

- **Developer Experience**: Improved documentation to make it easier for developers to:
  - Understand API authentication methods
  - Learn common API workflows
  - Get started quickly with code samples
  - Build their own client libraries

## Benefits

- Better developer onboarding experience
- Clear examples for common use cases
- Reduces time to first API call
- Provides templates for building client libraries
- Demonstrates best practices for API integration

## Testing

- Verified all code syntax is correct
- Checked that examples follow API specification
- Ensured consistent formatting and style