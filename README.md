# Strata

A Go web application starter/template project. Fork or copy this to create new applications.

## Features

- **Multi-auth login**: Trust (dev), Password, Email verification, Google OAuth
- **User management**: Admin user CRUD with enable/disable
- **Site settings**: Configurable site name, logo, footer
- **Dynamic pages**: Editable About, Contact, Terms, Privacy pages
- **Audit logging**: Auth events and admin actions
- **Dark mode**: User-selectable theme preference
- **Role-based access**: Admin role with extensible permissions

## Tech Stack

- **Go 1.24+** - Programming language
- **Chi** - HTTP router
- **MongoDB** - Database
- **HTMX** - Dynamic updates
- **Tailwind CSS** - Styling
- **Waffle** - Custom web framework (templates, storage, etc.)

## Getting Started

### Prerequisites

- Go 1.24 or later
- MongoDB running locally or accessible
- (Optional) Google OAuth credentials for Google login

### Configuration

Set environment variables (or use `.env` file):

```bash
# Required
export STRATA_MONGO_URI="mongodb://localhost:27017"
export STRATA_MONGO_DATABASE="strata"
export STRATA_SESSION_KEY="your-32-char-random-secret-key"

# Optional
export STRATA_PORT="3000"
export STRATA_ENV="development"  # or "production"

# Google OAuth (optional)
export STRATA_GOOGLE_CLIENT_ID="..."
export STRATA_GOOGLE_CLIENT_SECRET="..."

# Audit logging (optional)
export STRATA_AUDIT_AUTH="all"   # all, db, log, off
export STRATA_AUDIT_ADMIN="all"  # all, db, log, off
```

### Running

```bash
# Build and run
make run

# Or run in development mode
make dev

# Or directly
go run ./cmd/strata
```

The server starts at http://localhost:3000

### First Admin User

Create an admin user via environment variable on first run:

```bash
export STRATA_SEED_ADMIN_EMAIL="admin@example.com"
export STRATA_SEED_ADMIN_NAME="Admin User"
```

Or seed after building:

```bash
make seed-admin EMAIL=admin@example.com
```

## Project Structure

```
strata/
├── cmd/strata/          # Application entry point
├── internal/
│   ├── app/
│   │   ├── bootstrap/   # App lifecycle (config, db, routes, etc.)
│   │   ├── features/    # Feature handlers
│   │   │   ├── auditlog/
│   │   │   ├── dashboard/
│   │   │   ├── errors/
│   │   │   ├── health/
│   │   │   ├── home/
│   │   │   ├── login/
│   │   │   ├── logout/
│   │   │   ├── pages/
│   │   │   ├── profile/
│   │   │   ├── settings/
│   │   │   └── systemusers/
│   │   ├── resources/   # Templates and assets (embedded)
│   │   ├── store/       # MongoDB stores
│   │   └── system/      # Utilities (auth, viewdata, etc.)
│   └── domain/
│       └── models/      # Domain models
├── Makefile
└── README.md
```

## Extending Strata

### Adding a New Role

1. Add role constant in `system/authz/roles.go`
2. Update user validation in `store/users/userstore.go`
3. Create menu template `menu_<role>` in templates
4. Update menu dispatcher in layout
5. Create role-specific dashboard view

### Adding a New Feature

1. Create package in `features/<feature>/`
2. Implement Handler with routes
3. Add route mounting in `bootstrap/routes.go`
4. Create templates in `resources/templates/<feature>/`

## License

MIT
