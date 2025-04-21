# Log GUI (loggui)

A web interface and service to monitor and view logs from multiple GO http servers into one.

Goals of the project:

- Provide an easily hosted web server using net/http or any common web server.
- Separate logs between different request contexts
- Capability to store logs in a database starting with PostgreSQL and SQLite
- Re-use popular already implemented loggers

## Roadmap:

This is a roadmap of the project and what I plan to release in each stage

### Alpha

- [ ] Create a web server to receive requests via HTTP
- [ ] Create a client to send requests to the server. Start with the basics - HTTP status, Request Path, Time/Date
- [ ] Add interface for context-specific logs, and an adapter for Zap
- [ ] Create a basic UI with basic authentication and filters
- [ ] Add support for PostgreSQL and SQLite

### 1.0
T.B.D.
