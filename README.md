<div align="center">

# 🔍 CharityLens

### Transparent Charity Analysis for Informed Giving

*A powerful Go web application that helps donors make informed decisions by analyzing UK charity data from the Charity Commission for England and Wales.*

[![Go Version](https://img.shields.io/badge/Go-1.19+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-OGL%20v3.0-blue.svg)](https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/)
[![Database](https://img.shields.io/badge/Database-SQLite%20%7C%20MySQL%20%7C%20PostgreSQL-green.svg)](https://github.com)

[Features](#-features) • [Quick Start](#-quick-start) • [Documentation](#-documentation) • [API](#-api-reference) • [Development](#-development)

</div>

---

## 📊 Overview

**CharityLens** provides comprehensive transparency scoring and analysis for UK registered charities. Built with Go and modern web technologies, it delivers fast, accurate insights into charity performance, financial health, and governance.

### Why CharityLens?

- **Data-Driven Decisions**: Make informed donation choices based on objective metrics
- **Comprehensive Scoring**: Multi-dimensional analysis covering efficiency, financial health, transparency, and governance
- **Fast & Responsive**: Modern web interface with instant search results
- **Privacy-Focused**: No tracking, no user accounts, no data collection
- **Fully Open Source**: Transparent methodology and auditable code

---

## ✨ Features

### 🔎 **Smart Search**
Search 350,000+ UK charities by name, registration number, or cause area with instant results powered by HTMX.

### 📈 **Intelligent Scoring**
Composite transparency score (0-100) with confidence levels based on:
- **Efficiency (40%)**: How much goes to charitable programs vs. overhead
- **Financial Health (30%)**: Reserve adequacy, income trends, sustainability
- **Transparency (20%)**: Timely filing, data completeness, online presence
- **Governance (10%)**: Trustee structure, policies, and accountability

### 🔬 **Detailed Analysis**
View comprehensive charity profiles including:
- Financial metrics and trends
- Trustee information and governance
- Activities and impact areas
- Contact details and web presence
- Historical performance data

### ⚖️ **Side-by-Side Comparison**
Compare up to 5 charities simultaneously to find the best match for your values and giving priorities.

### 🌐 **REST API**
Full JSON API for integrations, research, and data analysis.

### 🚀 **Offline Mode**
Run completely offline with a pre-seeded database—no API key required for development or air-gapped deployments.

---

## 🚀 Quick Start

### Prerequisites

| Requirement | Version | Notes |
|-------------|---------|-------|
| **Go** | 1.19+ | [Download](https://go.dev/dl/) |
| **Database** | Any | SQLite (default), MySQL, or PostgreSQL |
| **API Key** | Optional | Only required for live sync mode ([Get one here](https://api-portal.charitycommission.gov.uk/)) |

### Installation

#### Option 1: Standard Installation (with API access)

```bash
# 1. Clone the repository
git clone https://github.com/yourusername/charitylens.git
cd charitylens

# 2. Install dependencies
go mod download

# 3. Build the application
go build -o charitylens ./cmd/charitylens

# 4. Run with your API key
./charitylens -api-key YOUR_API_KEY_HERE
```

Visit **http://localhost:8080** to start exploring charities!

#### Option 2: Offline Mode (no API key required)

Perfect for development, testing, or deployments without internet access:

```bash
# 1. Clone and build
git clone https://github.com/yourusername/charitylens.git
cd charitylens
go mod download
go build -o charitylens ./cmd/charitylens

# 2. Download pre-seeded database (247MB with 350k+ charities)
# See SEEDING.md for database creation instructions
cd cmd/charityseeder
go build -o charityseeder
./charityseeder -mode file  # Fast import from Charity Commission data dumps

# 3. Run in offline mode
cd ../..
./charitylens -offline -port 8080
```

Visit **http://localhost:8080** to explore the full database offline!

---

## ⚙️ Configuration

CharityLens supports both environment variables and command-line flags. **Flags override environment variables.**

### Environment Variables

```bash
# Database Configuration
export DATABASE_TYPE=sqlite              # sqlite | mysql | postgres
export DATABASE_URL=charitylens.db       # Path or connection string

# Server Configuration
export PORT=8080                         # HTTP port
export IP=0.0.0.0                        # Bind address

# API Configuration (standard mode only)
export CHARITY_API_KEY=your_api_key      # From Charity Commission portal
export SYNC_INTERVAL_HOURS=24            # Background sync frequency

# Development
export DEBUG=false                       # Enable detailed logging
export GO_ENV=development                # Hot-reload CSS/JS (no rebuild needed)
export OFFLINE_MODE=true                 # Run without API access
```

### Command-Line Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-port` | int | 8080 | HTTP port to bind to |
| `-ip` | string | 0.0.0.0 | IP address to bind to |
| `-api-key` | string | - | Charity Commission API subscription key |
| `-offline` | bool | false | Run in offline mode (no API calls) |
| `-debug` | bool | false | Enable detailed debug logging |

### Usage Examples

```bash
# Production mode with API access
./charitylens -port 3000 -api-key sub_abc123xyz

# Development mode with hot CSS/JS reload
GO_ENV=development ./charitylens -offline -port 8082

# Debug mode for troubleshooting
./charitylens -debug -api-key sub_abc123xyz

# Offline mode with custom database
DATABASE_URL=/data/charitylens.db ./charitylens -offline
```

**💡 Tip**: Visit http://localhost:8080 (or your configured address) to access the web interface.

---

## 🔌 Offline Mode

CharityLens can run completely **offline** using a pre-seeded database—no internet connection or API key required.

### When to Use Offline Mode

| Use Case | Benefit |
|----------|---------|
| **Development** | No API rate limits, instant results, reproducible data |
| **Testing** | Consistent test data, no external dependencies |
| **Air-Gapped Deployments** | Run on systems without internet access |
| **Data Snapshots** | Analyze charity data from a specific point in time |
| **Cost Savings** | No API costs, reduced bandwidth usage |

### Setting Up Offline Mode

**Step 1: Build the Seeder Tool**
```bash
cd cmd/charityseeder
go build -o charityseeder
```

**Step 2: Import Charity Data**

Choose from two import methods:

```bash
# Fast: Import all 350k+ charities in ~5-10 minutes
./charityseeder -mode file

# Selective: Import specific charity number ranges
./charityseeder -mode api -start 1 -end 10000 -api-key YOUR_KEY
```

See **[SEEDING.md](SEEDING.md)** for detailed seeding documentation.

**Step 3: Run CharityLens Offline**
```bash
cd ../..
./charitylens -offline
```

### Offline Mode Limitations

| Feature | Status | Notes |
|---------|--------|-------|
| Search & Browse | ✅ Full functionality | All data served from database |
| Charity Details | ✅ Full functionality | Includes scores, financials, trustees |
| Comparison | ✅ Full functionality | Compare any charities in database |
| API Endpoints | ✅ All except `/api/admin/sync` | Read-only operations work normally |
| Background Sync | ❌ Disabled | `/api/admin/sync` returns error |
| New Charities | ❌ No discovery | Limited to pre-seeded data |
| Live Updates | ❌ No refresh | Database is static snapshot |

---

## 🌐 Web Interface

CharityLens provides a modern, responsive web interface built with server-side HTML templates and HTMX for dynamic interactions.

### Pages

| Route | Description | Features |
|-------|-------------|----------|
| **`/`** | **Homepage & Search** | Hero section, live search with HTMX, filter chips, instant results |
| **`/charity/{number}`** | **Charity Details** | Transparency scores with animated rings, financials, trustees, activities, contact info |
| **`/compare`** | **Comparison Tool** | Side-by-side comparison of up to 5 charities with winner badges |
| **`/methodology`** | **Scoring Methodology** | Transparent documentation of scoring algorithm and data sources |
| **`/license`** | **Data License** | Open Government Licence v3.0 information |

### Design Features

- **Responsive Design**: Mobile-first, works on all screen sizes
- **Modern UI**: Gradients, glassmorphism effects, smooth animations
- **Accessible**: Keyboard navigation support, semantic HTML
- **Fast**: HTMX for instant interactions without full page reloads
- **Clean**: Minimal design focused on data and usability

---

## 🔗 API Reference

CharityLens provides a full REST API for integrations and programmatic access.

### Endpoints

#### Search Charities
```http
GET /api/charities/search?q={query}&limit={limit}
```

**Query Parameters:**
- `q` (required): Search query (name, number, or keywords)
- `limit` (optional): Max results to return (default: 20, max: 100)

**Response:**
```json
{
  "charities": [
    {
      "number": 1137606,
      "name": "Cancer Research UK",
      "status": "Registered",
      "website": "https://www.cancerresearchuk.org"
    }
  ],
  "count": 1
}
```

#### Get Charity Details
```http
GET /api/charities/{number}
```

**Parameters:**
- `number` (required): Charity registration number

**Response:**
```json
{
  "charity": {
    "number": 1137606,
    "name": "Cancer Research UK",
    "status": "Registered",
    "website": "https://www.cancerresearchuk.org",
    "income": 718000000,
    "spending": 695000000
  },
  "score": {
    "overall": 87,
    "efficiency": 92,
    "financial_health": 85,
    "transparency": 88,
    "governance": 81,
    "confidence": "high"
  },
  "trustees": [...],
  "activities": [...]
}
```

#### Compare Charities
```http
GET /api/charities/compare?numbers={numbers}
```

**Query Parameters:**
- `numbers` (required): Comma-separated charity numbers (max 5)

**Response:**
```json
{
  "charities": [
    {
      "number": 1137606,
      "name": "Cancer Research UK",
      "score": {...}
    },
    {
      "number": 205017,
      "name": "British Red Cross Society",
      "score": {...}
    }
  ]
}
```

#### Trigger Background Sync
```http
POST /api/admin/sync
```

**Notes:**
- Only available in standard mode (disabled in offline mode)
- Initiates background refresh from Charity Commission API
- Returns immediately; sync runs asynchronously

**Response:**
```json
{
  "message": "Sync initiated",
  "timestamp": "2025-12-29T10:30:00Z"
}
```

---

## 🗄️ Database Support

CharityLens is **database-agnostic** and supports SQLite, MySQL, and PostgreSQL through a unified abstraction layer.

### Configuration

Set the `DATABASE_TYPE` and `DATABASE_URL` environment variables based on your preferred database:

#### SQLite (Default - Recommended for Development)

Perfect for single-server deployments, development, and testing.

```bash
export DATABASE_TYPE=sqlite
export DATABASE_URL=charitylens.db
```

**Pros:**
- Zero configuration
- Single file database (portable)
- Fast for read-heavy workloads
- No separate database server needed

**Cons:**
- Not ideal for high-concurrency writes
- Limited to single server

#### MySQL

Great for production deployments with high traffic.

```bash
export DATABASE_TYPE=mysql
export DATABASE_URL=user:password@tcp(localhost:3306)/charitylens?parseTime=true
```

**Connection String Format:**
```
[username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
```

**Example:**
```bash
export DATABASE_URL=charitylens_user:securepassword@tcp(db.example.com:3306)/charitylens?parseTime=true&charset=utf8mb4
```

#### PostgreSQL

Ideal for advanced features and large-scale deployments.

```bash
export DATABASE_TYPE=postgres
export DATABASE_URL=postgres://user:password@localhost:5432/charitylens?sslmode=disable
```

**Connection String Format:**
```
postgres://[user[:password]@][host][:port][/dbname][?param1=value1&...&paramN=valueN]
```

**Example with SSL:**
```bash
export DATABASE_URL=postgres://charitylens_user:securepassword@db.example.com:5432/charitylens?sslmode=require
```

### Database Schema

CharityLens uses Go migrations for database schema management. The schema is automatically applied on first run and includes:

- **charities** - Core charity information
- **financials** - Income, spending, and reserve data
- **trustees** - Trustee and governance information
- **charity_scores** - Calculated transparency scores
- **activities** - Charity activities and cause areas
- **search_cache** - Search performance optimization
- **scraper_checkpoints** - Seeding progress tracking
- **linked_charities** - Parent/subsidiary relationships

See `migrations/` directory for full schema definitions.

---

## 🔄 Data Synchronization

CharityLens can sync live data from the **Charity Commission for England and Wales** API when running in standard mode.

### Getting an API Key

**Standard mode requires a Charity Commission API subscription key.** To obtain one:

1. **Register**: Visit [Charity Commission API Portal](https://api-portal.charitycommission.gov.uk/)
2. **Subscribe**: Subscribe to the "Register of Charities" API product
3. **Configure**: Set your API key via environment variable or flag:
   ```bash
   export CHARITY_API_KEY=sub_abc123xyz
   # OR
   ./charitylens -api-key sub_abc123xyz
   ```

### Sync Behavior

| Mode | API Key Required | Sync Behavior |
|------|------------------|---------------|
| **Standard** | ✅ Yes | On-demand sync when charities are requested |
| **Offline** | ❌ No | No syncing; serves pre-seeded database only |

### Background Sync

In standard mode, CharityLens refreshes stale charity data automatically:

- **Triggered by**: Search requests, charity detail views
- **Frequency**: Configurable via `SYNC_INTERVAL_HOURS` (default: 24 hours)
- **Manual Trigger**: POST to `/api/admin/sync` endpoint
- **Rate Limiting**: Built-in rate limiter respects API quotas

### Data Freshness

CharityLens tracks when each charity was last updated and displays data freshness warnings:

- **Fresh** (< 7 days): No warning
- **Stale** (7-30 days): "Data may be outdated" notice
- **Very Stale** (> 30 days): "Data needs refresh" warning

**💡 Tip**: Use offline mode for development to avoid API rate limits and ensure consistent test data.

---

## 🚢 Deployment

CharityLens is designed for flexible deployment options—from single-server setups to containerized environments.

### Docker Deployment

#### Quick Start with Docker

```bash
# Build the image
docker build -t charitylens:latest .

# Run with SQLite (database persisted in volume)
docker run -d \
  --name charitylens \
  -p 8080:8080 \
  -v charitylens-data:/app/data \
  -e DATABASE_URL=/app/data/charitylens.db \
  -e OFFLINE_MODE=true \
  charitylens:latest

# Run with external PostgreSQL
docker run -d \
  --name charitylens \
  -p 8080:8080 \
  -e DATABASE_TYPE=postgres \
  -e DATABASE_URL=postgres://user:pass@db-host:5432/charitylens \
  -e CHARITY_API_KEY=sub_abc123xyz \
  charitylens:latest
```

#### Docker Compose

```yaml
version: '3.8'

services:
  charitylens:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_TYPE=postgres
      - DATABASE_URL=postgres://charitylens:password@db:5432/charitylens
      - CHARITY_API_KEY=${CHARITY_API_KEY}
      - SYNC_INTERVAL_HOURS=24
    depends_on:
      - db
    restart: unless-stopped

  db:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=charitylens
      - POSTGRES_USER=charitylens
      - POSTGRES_PASSWORD=password
    volumes:
      - postgres-data:/var/lib/postgresql/data
    restart: unless-stopped

volumes:
  postgres-data:
```

### Single Binary Deployment

CharityLens compiles to a **single, self-contained binary** with no external dependencies (except database).

#### Build for Production

```bash
# Build with optimizations
go build -ldflags="-s -w" -o charitylens ./cmd/charitylens

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o charitylens-linux-amd64 ./cmd/charitylens
GOOS=darwin GOARCH=arm64 go build -o charitylens-darwin-arm64 ./cmd/charitylens
GOOS=windows GOARCH=amd64 go build -o charitylens-windows-amd64.exe ./cmd/charitylens
```

**Note**: When modifying embedded files (CSS/JS), rebuild with `-a` flag:
```bash
go build -a -o charitylens ./cmd/charitylens
```

#### Systemd Service (Linux)

Create `/etc/systemd/system/charitylens.service`:

```ini
[Unit]
Description=CharityLens Transparency Tool
After=network.target

[Service]
Type=simple
User=charitylens
WorkingDirectory=/opt/charitylens
ExecStart=/opt/charitylens/charitylens -api-key YOUR_KEY_HERE
Restart=on-failure
RestartSec=10

Environment="DATABASE_TYPE=sqlite"
Environment="DATABASE_URL=/opt/charitylens/data/charitylens.db"
Environment="PORT=8080"

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable charitylens
sudo systemctl start charitylens
sudo systemctl status charitylens
```

### Reverse Proxy (Nginx)

```nginx
server {
    listen 80;
    server_name charitylens.example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Performance Considerations

- **SQLite**: Great for < 100 concurrent users, single server deployments
- **MySQL/PostgreSQL**: Recommended for production with > 100 concurrent users
- **Pre-seeded Database**: Import all charities for instant search results
- **CDN**: Consider caching static assets (CSS, JS) via CDN
- **Memory**: ~200MB base + database cache (adjust based on dataset size)

---

## 📥 Database Seeding

CharityLens includes a powerful **seeding utility** to pre-populate your database with comprehensive charity data.

### Why Seed the Database?

- **Instant Results**: Search 350,000+ charities immediately without API delays
- **Offline Development**: Work without internet or API key
- **Consistent Testing**: Reproducible test data
- **Performance**: Sub-second search across entire UK charity register

### Two Import Modes

#### 🚀 File Mode (Recommended for Full Import)

Import **all 350,000+ UK charities in ~5-10 minutes** from Charity Commission data dumps.

```bash
# Step 1: Download official data dumps (updated weekly)
wget http://download.charitycommission.gov.uk/register/api/publicextract.charity.json
wget http://download.charitycommission.gov.uk/register/api/publicextract.charity_trustee.json

# Step 2: Build and run seeder
cd cmd/charityseeder
go build -o charityseeder
./charityseeder -mode file

# Result: ~247MB database with complete charity register
```

**What gets imported:**
- Core charity information (name, number, status, contact details)
- Financial data (income, spending, reserves)
- Trustee information
- Activities and cause areas
- Parent/subsidiary relationships

#### 🎯 API Mode (Selective Import)

Import **specific charity number ranges** directly from the Charity Commission API.

```bash
cd cmd/charityseeder
export CHARITY_API_KEY='sub_abc123xyz'
go build -o charityseeder

# Import charities 1-10,000
./charityseeder -mode api -start 1 -end 10000

# Resume from checkpoint if interrupted
./charityseeder -mode api -start 10001 -end 50000
```

**Use cases:**
- Testing with small datasets
- Updating specific charity ranges
- Incremental imports
- Development with limited data

### Advanced Seeding Options

```bash
# File mode with custom paths
./charityseeder -mode file \
  -charity-file /path/to/publicextract.charity.json \
  -trustee-file /path/to/publicextract.charity_trustee.json

# API mode with multiple keys (load balancing)
./charityseeder -mode api \
  -api-keys "key1,key2,key3" \
  -start 1 -end 100000

# Resume from checkpoint
./charityseeder -mode api -resume
```

### Performance Comparison

| Mode | Speed | Dataset | API Key Required | Use Case |
|------|-------|---------|------------------|----------|
| **File** | ⚡ 5-10 mins | All 350k+ charities | ❌ No | Production, full database |
| **API** | 🐢 Hours-days | Custom ranges | ✅ Yes | Development, selective import |

### Database Size Reference

| Dataset | Size | Charities | Includes |
|---------|------|-----------|----------|
| **Full Import** | ~247 MB | 354,754 | All charities (active + removed) |
| **Active Only** | ~140 MB | 171,232 | Active/registered charities only |
| **Sample (10k)** | ~7 MB | 10,000 | Testing dataset |

📖 **For complete seeding documentation**, see **[SEEDING.md](SEEDING.md)** including:
- Detailed performance benchmarks
- Resume and checkpoint management
- Multi-key load balancing
- Troubleshooting common issues

---

## 💻 Development

### Project Structure

```
charitylens/
├── cmd/
│   ├── charitylens/              # Main application entry point
│   │   └── main.go
│   └── charityseeder/            # Database seeding utility
│       └── main.go
├── internal/
│   ├── api/                      # Charity Commission API client
│   │   ├── client.go             # HTTP client with rate limiting
│   │   ├── parser.go             # Response parsing and validation
│   │   └── ratelimiter.go        # Token bucket rate limiter
│   ├── config/                   # Configuration management
│   │   └── config.go             # Environment and flag parsing
│   ├── database/                 # Database abstraction layer
│   │   └── db.go                 # SQLite, MySQL, PostgreSQL support
│   ├── errors/                   # Custom error types
│   │   └── errors.go             # Application error definitions
│   ├── handlers/                 # HTTP request handlers
│   │   ├── charities.go          # API endpoints
│   │   └── web.go                # Web page handlers
│   ├── importer/                 # Data import logic
│   │   └── importer.go           # File and API import
│   ├── logger/                   # Logging utilities
│   │   └── logger.go             # Structured logging
│   ├── middleware/               # HTTP middleware
│   │   └── middleware.go         # Logging, recovery, CORS
│   ├── models/                   # Data models and structures
│   │   └── models.go             # Charity, Score, Trustee types
│   ├── scoring/                  # Transparency scoring algorithm
│   │   └── scoring.go            # Multi-dimensional scoring logic
│   ├── sync/                     # Background data synchronization
│   │   └── sync.go               # Periodic API sync worker
│   └── version/                  # Version information
│       └── version.go            # Build version and metadata
├── migrations/                   # Database schema migrations
│   ├── 001_create_charities_table.up.sql
│   ├── 002_create_financials_table.up.sql
│   ├── 003_create_trustees_table.up.sql
│   ├── 004_create_charity_scores_table.up.sql
│   └── ... (and corresponding .down.sql files)
├── web/
│   ├── static/                   # Static assets
│   │   ├── static.go             # Embedded files (CSS, JS)
│   │   ├── css/
│   │   │   └── main.css          # ~2,000 lines of modern CSS
│   │   └── js/
│   │       └── main.js           # Frontend JavaScript
│   └── templates/                # HTML templates
│       ├── templates.go          # Template parsing and functions
│       ├── header.html           # Navigation partial
│       ├── footer.html           # Footer partial
│       ├── index.html            # Homepage and search
│       ├── charity.html          # Charity detail page
│       ├── compare.html          # Comparison page
│       ├── methodology.html      # Scoring documentation
│       ├── license.html          # Data license info
│       └── error.html            # Error page
├── .dockerignore
├── .gitignore
├── Dockerfile                    # Container image definition
├── go.mod                        # Go module dependencies
├── go.sum                        # Dependency checksums
├── README.md                     # This file
└── SEEDING.md                    # Database seeding documentation
```

### Development Workflow

#### Local Development

```bash
# 1. Start development server with hot CSS/JS reload
GO_ENV=development ./charitylens -offline -port 8082

# 2. Make changes to CSS/JS files
# Changes are immediately visible (no rebuild needed)

# 3. For code changes, rebuild and restart
go build -o charitylens ./cmd/charitylens
GO_ENV=development ./charitylens -offline -port 8082
```

#### Important Build Notes

**Embedded Files**: CSS and JS are embedded in the binary via `//go:embed` directives.

- **Production Build**: Embeds current CSS/JS snapshot
  ```bash
  go build -a -o charitylens ./cmd/charitylens
  ```
  
- **Development Mode**: Serves CSS/JS from disk (hot reload)
  ```bash
  GO_ENV=development ./charitylens -offline
  ```

**When to use `-a` flag:**
- After modifying CSS or JS files (forces rebuild of embedded assets)
- When Go build cache causes stale embedded files
- For production releases

**Normal builds** (no `-a`):
```bash
go build -o charitylens ./cmd/charitylens
```

#### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Test specific package
go test ./internal/scoring/
```

#### Code Quality

```bash
# Vet code for issues
go vet ./...

# Format code
go fmt ./...

# Run linter (requires golangci-lint)
golangci-lint run

# Check for security issues (requires gosec)
gosec ./...
```

#### Database Migrations

CharityLens uses numbered SQL migrations in the `migrations/` directory.

**Creating new migrations:**

```bash
# Create new migration files (manually)
touch migrations/011_add_feature.up.sql
touch migrations/011_add_feature.down.sql
```

**Migration naming convention:**
- `XXX_description.up.sql` - Apply migration
- `XXX_description.down.sql` - Rollback migration
- XXX = zero-padded sequence number (001, 002, 003...)

### Contributing Guidelines

1. **Fork and Clone**: Fork the repository and clone your fork
2. **Create Branch**: Create a feature branch (`git checkout -b feature/amazing-feature`)
3. **Write Tests**: Add tests for new functionality
4. **Format Code**: Run `go fmt ./...` before committing
5. **Commit**: Write clear commit messages following conventional commits
6. **Test**: Ensure all tests pass (`go test ./...`)
7. **Push**: Push to your fork and open a pull request
8. **Document**: Update README.md and SEEDING.md if needed

### Development Tips

- **Use Offline Mode**: Avoid API rate limits during development
- **Pre-seed Database**: Import full dataset for realistic testing
- **Watch Logs**: Use `-debug` flag for detailed logging
- **Hot Reload CSS**: Use `GO_ENV=development` for instant CSS changes
- **Test Databases**: SQLite is fastest for development; test production DB before deploying

---

## 🧮 Scoring Methodology

CharityLens calculates a **composite transparency score (0-100)** using four weighted dimensions:

### Score Components

| Component | Weight | What It Measures |
|-----------|--------|------------------|
| **Efficiency** | 40% | Program spending ratio (charitable activities ÷ total spending) |
| **Financial Health** | 30% | Reserve adequacy (3-12 months optimal), income trends, sustainability |
| **Transparency** | 20% | Timely filing, data completeness, web presence, public reporting |
| **Governance** | 10% | Trustee structure, policies, accountability mechanisms |

### Confidence Levels

CharityLens assigns confidence levels based on data quality and freshness:

- **High**: Complete recent data (< 1 year old), all fields populated
- **Medium**: Some missing data or slightly outdated (1-2 years old)
- **Low**: Significant missing data or very outdated (> 2 years old)

### Fair Scoring Principles

1. **No Editorial Bias**: Scoring is purely algorithmic
2. **Size-Neutral**: Small charities aren't penalized for lower overheads
3. **Context-Aware**: Different charity types have different optimal metrics
4. **Transparent**: Full methodology documented at `/methodology`
5. **Graceful Degradation**: Missing data doesn't break scores

📖 **For detailed scoring formulas and examples**, visit `/methodology` in the web interface or see `internal/scoring/scoring.go`.

---

## 🗺️ Roadmap

Future enhancements under consideration:

- [ ] **Advanced Search**: Filter by cause area, location, income range
- [ ] **Visualizations**: Charts for financial trends, spending breakdowns
- [ ] **Alerts**: Notify when favorite charities update their data
- [ ] **Exports**: Download comparison data as CSV/PDF
- [ ] **Historical Tracking**: Compare charity performance over time
- [ ] **User Accounts**: Save favorite charities and comparisons
- [ ] **Public API**: Rate-limited public API with authentication
- [ ] **Mobile App**: Native iOS/Android applications
- [ ] **Internationalization**: Support for non-English languages

**Have ideas?** Open an issue or submit a pull request!

---

## 🤝 Contributing

We welcome contributions from the community! Whether it's bug fixes, new features, documentation improvements, or feedback, all contributions are valued.

### How to Contribute

1. **Report Bugs**: Open an issue with detailed reproduction steps
2. **Suggest Features**: Describe your idea and use case
3. **Submit Code**: Fork, branch, code, test, and submit a PR
4. **Improve Docs**: Help make documentation clearer and more comprehensive
5. **Share Feedback**: Tell us what works well and what could be better

### Development Setup

See the [Development](#-development) section above for setup instructions.

### Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Assume good intentions
- Help create a welcoming environment

---

## 📄 License

CharityLens is open source software licensed under the **Open Government Licence v3.0**.

### Data License

Charity data is sourced from the **Charity Commission for England and Wales** and is subject to the [Open Government Licence v3.0](https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/).

**You are free to:**
- Copy, publish, distribute, and transmit the data
- Adapt the data
- Exploit the data commercially

**You must:**
- Acknowledge the source (Charity Commission for England and Wales)
- Provide a link to the Open Government Licence
- State if you've modified the data

**You cannot:**
- Use the data in a way that suggests official endorsement
- Mislead others about the source of the data

### Software License

The CharityLens application code is also licensed under the **Open Government Licence v3.0**, making it freely available for use, modification, and distribution.

---

## 🙏 Acknowledgments

- **Charity Commission for England and Wales** - For providing comprehensive open data
- **Go Community** - For excellent tooling and libraries
- **Open Source Contributors** - For dependencies and inspiration

---

## 📞 Support

### Documentation

- **README.md** - This file (overview and setup)
- **SEEDING.md** - Database seeding guide
- **[AGENTS.md](AGENTS.md)** - Build instructions and architecture
- **Web Interface** - `/methodology` and `/license` pages

### Getting Help

- **Issues**: Report bugs or request features via GitHub Issues
- **Discussions**: Ask questions and share ideas in GitHub Discussions
- **Documentation**: Check the docs above for detailed guidance

### Frequently Asked Questions

**Q: Do I need an API key?**  
A: No, if you use offline mode with a pre-seeded database. Yes, for standard mode with live data sync.

**Q: How often is charity data updated?**  
A: In standard mode, data refreshes automatically based on `SYNC_INTERVAL_HOURS` (default: 24 hours). In offline mode, data is static until you re-import.

**Q: Can I use this for commercial purposes?**  
A: Yes, both the software and data are licensed under the Open Government Licence v3.0, which permits commercial use.

**Q: How accurate are the transparency scores?**  
A: Scores are algorithmically calculated from official Charity Commission data. Accuracy depends on data quality and freshness. Always check the confidence level and data timestamp.

**Q: Does this work for charities outside England and Wales?**  
A: No, CharityLens currently only supports charities registered with the Charity Commission for England and Wales. Scottish charities (OSCR) and Northern Irish charities are not included.

**Q: How can I contribute?**  
A: See the [Contributing](#-contributing) section above!

---

<div align="center">

**Made with ❤️ for informed charitable giving**

[⬆ Back to Top](#-charitylens)

</div>