# Changelog

All notable changes to the `components/core/manageai` package will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-03-20

### Added

- **Core Client Functionality**
  - Complete ManageAI API client implementation
  - HTTP client with configurable timeouts and retry mechanisms
  - Support for all major AI service endpoints

- **Chat Completions**
  - Full chat completion API support
  - Streaming chat completion with Server-Sent Events (SSE)
  - Comprehensive message types (system, human, assistant, tool)
  - Multi-modal message support with image content
  - Extensive chat completion options configuration

- **Tokenization Services**
  - Text tokenization capabilities
  - Token streaming support
  - Integration with core client functionality

- **Embedding Services**
  - Text embedding generation
  - Batch processing support
  - Multiple encoding format options

- **Reranking Services**
  - Document relevance scoring
  - Intelligent result ranking
  - Configurable reranking options

- **Detokenization Services**
  - Text detokenization from token arrays
  - Model-specific detokenization support
  - Integration with existing tokenization workflow

- **Model Management**
  - Model listing and validation
  - Rate limit inspection and monitoring
  - Token usage tracking
  - Health and version checking

- **Advanced Features**
  - Smart model aliasing and redirection
  - Comprehensive error handling with custom error types
  - Struct-to-Tool conversion for automatic JSON schema generation
  - Tool call argument parsing and validation
  - JSON Schema property enhancement methods
  - Rate limit and quota monitoring

- **Message Types**
  - System messages for AI context setting
  - Human/User messages for input
  - Assistant messages for AI responses
  - Tool messages for function call results
  - Multi-modal content blocks for rich media

- **Configuration**
  - Builder pattern for chat completion options
  - Environment-based client initialization
  - Customizable timeout and retry settings

- **Testing**
  - Comprehensive test coverage for all features
  - Integration tests for real API endpoints
  - Streaming functionality verification

### Changed

- **Documentation Improvements**
  - Complete API documentation with examples
  - Quick start guides for all major features
  - Detailed usage instructions for advanced features

### Fixed

- **Initial Release Stability**
  - All core functionality tested and verified
  - Error handling for edge cases implemented
  - Memory management and connection cleanup ensured

## [0.1.0] - 2026-02-15

### Added

- **Initial Development Release**
  - Basic client structure and foundation
  - Core message types implementation
  - Initial API endpoint integrations