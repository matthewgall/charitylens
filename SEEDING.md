# Creating a Seed Database for CharityLens

This guide explains how to use the `charityseeder` utility to create a pre-populated SQLite database of UK charity data.

## Why Seed a Database?

Instead of starting with an empty database and fetching charity data on-demand, you can:
- Pre-populate a database with thousands of charities
- **Pre-calculate transparency scores** for instant display (no computation at runtime)
- Distribute the database with your application
- Enable offline development and testing
- Reduce API calls during production use
- Speed up initial application startup

## Import Modes

The `charityseeder` supports **three modes** for populating and updating your database:

### 1. File Mode (FAST - Recommended)
Import from Charity Commission JSON data dumps. This is **much faster** than API mode and doesn't require an API key.

**Pros:**
- ⚡ **Extremely fast** - Import 395k charities in ~5-10 minutes
- 🔓 **No API key required**
- 📦 **Complete dataset** - All UK charities at once
- 💾 **Offline-friendly** - No internet needed after download

**Cons:**
- 📥 Requires downloading 745MB of JSON files first
- 📅 Data freshness depends on when you download

### 2. API Mode (Flexible)
Scrape directly from the Charity Commission API.

**Pros:**
- 🎯 **Selective import** - Choose specific charity number ranges
- 🔄 **Always current** - Get real-time data
- 📊 **Detailed data** - Includes financial breakdowns and trustees

**Cons:**
- 🔑 Requires API key(s)
- ⏱️ **Much slower** - 170k charities takes ~4-8 hours
- 🌐 Requires internet connection

### 3. Score Mode (Calculate/Recalculate)
Calculate transparency scores for charities already in the database.

**Use cases:**
- 🔄 **Resume interrupted imports** - If import was stopped before scoring completed
- 📊 **Recalculate scores** - After methodology changes
- 🆕 **Score new charities** - After adding charities via API or manual import

**Example:**
```bash
./charityseeder -mode score -db charitylens.db
```

This mode processes all charities without scores (~235 scores/second) and is safe to run multiple times.

## Quick Start

### File Mode (Fastest)

```bash
# 1. Download Charity Commission data dumps
wget http://download.charitycommission.gov.uk/register/api/publicextract.charity.json
wget http://download.charitycommission.gov.uk/register/api/publicextract.charity_trustee.json

# 2. Build and run in file mode
cd cmd/charityseeder
go build
./charityseeder -mode file

# That's it! All 395k charities imported in minutes.
```

### API Mode (For selective imports)

```bash
# 1. Get your API key(s) from https://developer.charitycommission.gov.uk/
export CHARITY_API_KEY='your-api-key-here'

# Or use multiple keys for better performance:
export CHARITY_API_KEYS='key1,key2,key3'

# 2. Build and run the seeder
cd cmd/charityseeder
go build
./charityseeder -mode api -start 1 -end 1000
```

## File Mode Usage

### Step 1: Download Charity Commission Data Dumps

The Charity Commission provides complete data dumps that you can download:

```bash
# Download charity data (481MB - ~395k charities)
wget http://download.charitycommission.gov.uk/register/api/publicextract.charity.json

# Download trustee data (264MB - ~923k trustee records)
wget http://download.charitycommission.gov.uk/register/api/publicextract.charity_trustee.json
```

Alternatively, download from the browser:
- Charity data: http://download.charitycommission.gov.uk/register/api/publicextract.charity.json
- Trustee data: http://download.charitycommission.gov.uk/register/api/publicextract.charity_trustee.json

### Step 2: Import the Data

```bash
cd cmd/charityseeder
go build

# Import all data (default file names)
./charityseeder -mode file

# Custom file paths
./charityseeder -mode file \
  -charity-file /path/to/publicextract.charity.json \
  -trustee-file /path/to/publicextract.charity_trustee.json

# Custom database location
./charityseeder -mode file -db /path/to/charitylens.db

# Adjust batch size for performance tuning
./charityseeder -mode file -batch-size 5000

# Enable verbose logging
./charityseeder -mode file -verbose
```

### Expected Output (File Mode)

```
=== File Import Mode ===
Charity file: publicextract.charity.json
Trustee file: publicextract.charity_trustee.json
Batch size: 1000

[1/4] Importing charities...
Starting charity import from: publicextract.charity.json
Progress: 5000 processed (4847 success, 12 failed, 141 skipped) | Rate: 850.32/sec
Progress: 10000 processed (9693 success, 23 failed, 284 skipped) | Rate: 862.15/sec
...
=== Charity import Complete ===
Total Records: 395024
Processed: 395024
Successful: 383142
Failed: 234
Skipped: 11648
Time Elapsed: 7m32s
Average Rate: 873.45 records/second

[2/4] Importing trustees...
Starting trustee import from: publicextract.charity_trustee.json
Progress: 5000 processed (4998 success, 1 failed, 1 skipped) | Rate: 920.45/sec
...
=== Trustee import Complete ===
Total Records: 923035
Processed: 923035
Successful: 922156
Failed: 89
Skipped: 790
Time Elapsed: 15m18s
Average Rate: 1005.23 records/second

[3/4] Importing detailed financial data...
Starting financial data import from: publicextract.charity_annual_return_partb.json
Progress: 5000 processed (4512 success, 8 failed, 480 skipped) | Rate: 780.12/sec
...
=== Financial data import Complete ===
Total Records: 421567
Processed: 421567
Successful: 395142
Failed: 112
Skipped: 26313
Time Elapsed: 9m45s
Average Rate: 720.34 records/second

[4/4] Calculating scores for all charities...
Found 383142 charities needing score calculation
Starting score calculation for all charities...
Progress: 5000 processed (4998 success, 2 failed, 0 skipped) | Rate: 645.23/sec
Progress: 10000 processed (9995 success, 5 failed, 0 skipped) | Rate: 652.18/sec
...
=== Score calculation Complete ===
Total Records: 383142
Processed: 383142
Successful: 382987
Failed: 155
Skipped: 0
Time Elapsed: 9m58s
Average Rate: 641.05 records/second

=== File Import Complete ===
```

### Performance Tips (File Mode)

- **Batch size**: Default 1000 works well. Increase to 5000 for faster imports on powerful machines
- **SSD storage**: Use SSD for the database file for better write performance
- **Memory**: File mode uses streaming JSON parsing, so memory usage stays low (~200MB)

### Score Precomputation

The file import mode automatically calculates transparency scores for **all imported charities** after the data is loaded. This means:

- **Instant search results**: Scores are already computed when users search
- **No runtime overhead**: Web requests don't need to calculate scores on-demand
- **Consistent scoring**: All charities scored with the same methodology at import time
- **Ready for production**: Database is immediately usable without additional processing

The scoring calculation happens in the `[4/4]` step and processes charities in batches of 1000, typically taking about 10 minutes for all 395k charities. Each score includes:
- **Efficiency Score** (40%): Ratio of charitable activities to total spending
- **Financial Health Score** (30%): Reserve adequacy (3-12 months optimal)
- **Transparency Score** (20%): Website presence, financial data, trustee disclosure
- **Governance Score** (10%): Trustee count and structure
- **Overall Score**: Weighted composite (0-100)
- **Confidence Level**: High/medium/low based on data completeness and freshness

## API Mode Usage

### Build the Seeder

```bash
cd cmd/charityseeder
go build
```

### Basic Commands

**Test with 100 charities** (great for testing):
```bash
./charityseeder -mode api -start 1 -end 100 -verbose
```

**Scrape 10,000 charities** (good seed size):
```bash
./charityseeder -mode api -start 1 -end 10000

# Output:
# 🔍 Starting scraper
#    Range: 1 to 10000
#    Rate limit: 10 req/s
#    Workers: 5
#    API keys: 1
#
# [1/3] Scraping charities...  45% [==================>                    ] (4523/10000, 12 charities/s) [6m:7m]
```

**Full scrape** (all ~170,000 UK charities):
```bash
./charityseeder -mode api -rate-limit 5 -concurrency 3
```

**Note:** For importing ALL charities, file mode is much faster (minutes vs hours).

### Resume After Interruption (API Mode Only)

The API mode automatically saves progress every 100 charities. If interrupted, simply run again:

```bash
./charityseeder -mode api
```

It will resume from the last checkpoint.

## Mode Comparison

| Feature | File Mode | API Mode |
|---------|-----------|----------|
| **Speed** | ⚡ 5-10 minutes for all 395k | ⏱️ 4-8 hours for 170k |
| **API Key** | ❌ Not required | ✅ Required |
| **Data Freshness** | Depends on download date | Real-time |
| **Selective Import** | ❌ All or nothing | ✅ Choose ranges |
| **Internet Required** | Only for download | Yes, continuously |
| **Trustee Data** | ✅ Included | ✅ Included |
| **Financial Details** | Basic (income/expenditure) | Detailed breakdowns |
| **Resumable** | ❌ Restart from beginning | ✅ Checkpoint every 100 |

## Recommended Seed Sizes

### Small Seed (100-1,000 charities) - API MODE
- **Time**: 1-5 minutes
- **Size**: ~1-5 MB
- **Use case**: Development, testing
- **Command**: `./charityseeder -mode api -start 1 -end 1000`

### Medium Seed (10,000 charities) - API MODE
- **Time**: 30-60 minutes
- **Size**: ~30-50 MB
- **Use case**: Demo deployments, local testing
- **Command**: `./charityseeder -mode api -start 1 -end 10000`

### Large Seed (50,000 charities) - API MODE
- **Time**: 2-4 hours
- **Size**: ~150-200 MB
- **Use case**: Production seed, comprehensive coverage
- **Command**: `./charityseeder -mode api -start 1 -end 50000`

### Full Seed (ALL 395k charities) - FILE MODE (RECOMMENDED)
- **Time**: ⚡ 15-25 minutes (includes score calculation)
- **Size**: ~500 MB - 1 GB
- **Use case**: Complete UK charity database with pre-computed scores
- **Command**: `./charityseeder -mode file`

### Full Seed (API Mode Alternative)
- **Time**: 4-8 hours
- **Size**: ~500 MB - 1 GB
- **Use case**: Need real-time data or selective ranges
- **Command**: `./charityseeder -mode api -rate-limit 5 -concurrency 3`

## Configuration Options

### File Mode Options

```bash
# Custom file paths
./charityseeder -mode file \
  -charity-file /path/to/publicextract.charity.json \
  -trustee-file /path/to/publicextract.charity_trustee.json

# Custom database location
./charityseeder -mode file -db /path/to/charitylens.db

# Adjust batch size (default 1000)
./charityseeder -mode file -batch-size 5000

# Verbose logging
./charityseeder -mode file -verbose
```

### API Mode Options

#### Rate Limiting

Control API politeness with `-rate-limit`:

```bash
# Very polite (5 req/s) - recommended for overnight runs
./charityseeder -mode api -rate-limit 5 -concurrency 2

# Balanced (10 req/s) - default
./charityseeder -mode api -rate-limit 10 -concurrency 5

# Faster (20 req/s) - use with caution
./charityseeder -mode api -rate-limit 20 -concurrency 10
```

#### Custom Ranges

Scrape specific charity number ranges:

```bash
# Major charities (typically lower numbers)
./charityseeder -mode api -start 1 -end 100000

# Recent charities
./charityseeder -mode api -start 900000 -end 999999

# Specific range
./charityseeder -mode api -start 250000 -end 350000
```

### Common Options (Both Modes)

#### Database Location

Specify where to save the database:

```bash
# File mode
./charityseeder -mode file -db /path/to/seed.db

# API mode
./charityseeder -mode api -db /path/to/seed.db
```

#### Migrations Path

By default, the seeder looks for migrations at `../../migrations` (relative to `cmd/charityseeder/`). If you're running from a different location or have migrations elsewhere, specify the path:

```bash
./charityseeder -mode file -migrations /path/to/migrations
./charityseeder -mode api -migrations /path/to/migrations
```

### Multiple API Keys (Load Balancing - API Mode)

For better performance and to avoid rate limits, you can use multiple API keys. The seeder will automatically distribute requests across all keys using round-robin:

```bash
# Using environment variable (comma-separated)
export CHARITY_API_KEYS='key1,key2,key3'
./charityseeder -mode api

# Or using command-line flag
./charityseeder -mode api -api-keys 'key1,key2,key3'
```

**Benefits of multiple keys:**
- Higher effective rate limit (each key has its own limit)
- Automatic failover if one key hits rate limit
- Load distribution across keys
- Stats showing usage per key

**Note:** File mode doesn't use API keys, so this only applies to API mode.

## Using the Seed Database

Once created, you can use the seed database with CharityLens:

### Option 1: Replace the Default Database

```bash
# Copy seed to main application
cp cmd/charityseeder/seed.db charitylens.db

# Run CharityLens
DATABASE_TYPE=sqlite DATABASE_URL=charitylens.db ./charitylens
```

### Option 2: Distribute with Application

Include the seed database in your distribution:

```bash
# Package for distribution
tar -czf charitylens-with-seed.tar.gz \
    charitylens \
    seed.db \
    migrations/ \
    web/
```

Users can then start with:
```bash
DATABASE_TYPE=sqlite DATABASE_URL=seed.db ./charitylens
```

### Option 3: Docker

Include in your Dockerfile:

```dockerfile
FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN cd cmd/charityseeder && go build

# Create seed database
ENV CHARITY_API_KEY=your-key
RUN cd cmd/charityseeder && ./charityseeder -start 1 -end 10000

FROM alpine:latest
COPY --from=builder /app/cmd/charityseeder/seed.db /app/
COPY --from=builder /app/charitylens /app/
CMD ["/app/charitylens"]
```

## Monitoring Progress

The seeder outputs progress every 30 seconds:

```
Progress: 312 processed (305 success, 1 failed, 6 skipped) | Current: 312 | Rate: 5.20/s | API: 298/min, 5/sec
```

This shows:
- **312 processed**: Total charities attempted
- **305 success**: Successfully stored in database
- **1 failed**: Failed after all retries
- **6 skipped**: Already existed in database
- **Current: 312**: Currently processing charity #312
- **Rate: 5.20/s**: Processing 5.2 charities per second
- **API: 298/min, 5/sec**: API request rate

## Handling Errors

### Rate Limited (429 errors)

If you see many 429 errors, reduce the rate:

```bash
./charityseeder -rate-limit 5 -concurrency 2
```

### Network Timeouts

Increase retry attempts:

```bash
./charityseeder -max-retries 10
```

### Database Locked

Reduce concurrency:

```bash
./charityseeder -concurrency 1
```

## Database Schema

The seeder uses the **same migration system** as the main CharityLens application (in `migrations/`), ensuring complete schema compatibility. The database is automatically initialized with all migrations when you run the seeder.

### Tables Created
- `charities` - Basic charity information (migration 001 + 007 for company_number)
- `financials` - Financial data for each charity (migration 002)
- `trustees` - Trustee names (migration 003)
- `charity_scores` - Calculated transparency scores (migration 004)
- `activities` - Charity activities (migration 005)
- `search_cache` - Search result caching (migration 008)
- `scraper_checkpoints` - Resume state for seeder (migration 009)

### Indexes
All indexes defined in the migrations are automatically created, including:
- `idx_charities_name` - Fast charity name lookup
- `idx_charities_status` - Filter by status
- `idx_financials_charity` - Financial data lookup
- `idx_trustees_charity` - Trustee lookup

The seed database is fully compatible with the main CharityLens application and can be used directly without any additional setup.

## Best Practices

### For Development
1. Use a small seed (100-1,000 charities)
2. Enable verbose logging: `-verbose`
3. Test with known charities: `-start 200000 -end 200100`

### For Testing
1. Use medium seed (10,000 charities)
2. Include variety of charity sizes
3. Refresh periodically (monthly)

### For Production
1. Use large seed (50,000+)
2. Conservative rate limits: `-rate-limit 5`
3. Run during off-hours
4. Keep seed updated (quarterly)

### For Distribution
1. Create seed database
2. Compress: `gzip seed.db`
3. Document version and date
4. Provide checksum: `sha256sum seed.db.gz`

## Performance Tips

### Faster Scraping
- **Use multiple API keys**: `-api-keys 'key1,key2,key3'` (best option!)
- Increase concurrency: `-concurrency 10`
- Increase rate limit: `-rate-limit 20`
- Use SSD for database

### More Polite
- Decrease rate limit: `-rate-limit 5`
- Decrease concurrency: `-concurrency 2`
- Run during off-peak hours

### Multiple API Keys Performance

With multiple API keys, you can achieve significantly higher throughput:

| Keys | Effective Rate | Charities/Hour | Time for 170k |
|------|---------------|----------------|---------------|
| 1    | 10 req/s      | ~36,000        | ~5 hours      |
| 2    | 20 req/s      | ~72,000        | ~2.5 hours    |
| 3    | 30 req/s      | ~108,000       | ~1.5 hours    |
| 5    | 50 req/s      | ~180,000       | ~1 hour       |

**Example with 3 keys:**
```bash
export CHARITY_API_KEYS='key1,key2,key3'
./charityseeder -concurrency 10
```

### Resume Strategies
- Let it run overnight
- Use `screen` or `tmux` for long sessions
- Monitor with verbose mode initially

## Troubleshooting

### "API key is required"
```bash
export CHARITY_API_KEY='your-key-here'
./charityseeder
```

### Build errors
```bash
cd cmd/charityseeder
go build
```

Note: The seeder uses the parent module (`charitylens`), so there's no separate `go.mod` in the seeder directory.

### Permission errors
```bash
chmod +x charityseeder
```

### Database corruption
```bash
rm seed.db
./charityseeder  # Start fresh
```

## Example Session

```bash
$ export CHARITY_API_KEY='abc123...'
$ cd cmd/charityseeder
$ go build
$ ./charityseeder -start 1 -end 5000 -verbose

2025/12/28 10:00:00 Starting scraper: charities 1 to 5000, rate limit: 10 req/s, concurrency: 5
2025/12/28 10:00:30 Progress: 145 processed (142 success, 0 failed, 3 skipped) | Current: 145 | Rate: 4.83/s
2025/12/28 10:01:00 Progress: 312 processed (305 success, 1 failed, 6 skipped) | Current: 312 | Rate: 5.20/s
...
2025/12/28 10:15:23 Progress: 4987 processed (4842 success, 12 failed, 133 skipped) | Current: 4987 | Rate: 5.42/s

=== Final Statistics ===
Total Processed: 5000
Successful: 4847
Failed: 12
Skipped: 141
Time Elapsed: 15m23s
Average Rate: 5.42 charities/second
Last Charity: 5000

$ ls -lh seed.db
-rw-r--r-- 1 user user 18M Dec 28 10:15 seed.db
```

## Next Steps

After creating your seed database:

1. **Test the database**:
   ```bash
   sqlite3 seed.db "SELECT COUNT(*) FROM charities;"
   ```

2. **Verify scores** (all scores are pre-calculated during import):
   ```bash
   # Check that scores were calculated
   sqlite3 seed.db "SELECT COUNT(*) FROM charity_scores;"
   
   # Run CharityLens with the seed database
   DATABASE_TYPE=sqlite DATABASE_URL=cmd/charityseeder/seed.db go run cmd/charitylens/main.go
   ```

3. **Distribute**:
   ```bash
   # Compress for distribution
   gzip -k seed.db
   sha256sum seed.db.gz > seed.db.gz.sha256
   ```

## Technical Details

### Implementation

The seeder is a standalone Go application that:
- Uses the shared database package and migrations from the main application
- Implements token bucket rate limiting for polite API usage
- Provides exponential backoff retry logic (1s → 2s → 4s → 8s → 16s)
- Uses a worker pool pattern for concurrent scraping
- Automatically saves checkpoints every 100 charities for resumability
- Handles graceful shutdown on interrupt signals (Ctrl+C)

### API Politeness Features

- **Rate Limiting**: Configurable requests per second (default: 10 req/s)
- **Exponential Backoff**: Automatic retry with increasing delays
- **Retry-After Respect**: Honors API rate limit headers
- **Proper User-Agent**: Identifies as "CharityLens-Seeder/1.0"
- **Request Tracking**: Monitors actual API request rates

### Data Quality

The seeder:
- Fetches complete charity details from the Charity Commission API
- Stores data in normalized tables with proper foreign keys
- Handles missing data gracefully (NULL values where appropriate)
- Skips charities that already exist in the database
- Uses transactions to ensure data integrity

## Additional Resources

- [Charity Commission API Documentation](https://developer.charitycommission.gov.uk/)
- [SQLite Documentation](https://www.sqlite.org/docs.html)
- [CharityLens Main Documentation](README.md)
- [Project Migrations](migrations/) - Single source of truth for database schema

## Support

For issues or questions:
1. Check the troubleshooting section above
2. Review logs with `-verbose` flag
3. Check that migrations are accessible (seeder needs `../../migrations/`)
4. Open an issue on GitHub
