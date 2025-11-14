#!/usr/bin/env bash

################################################################################
# PostgreSQL Cluster Manager Script
#
# Description: Manage multiple PostgreSQL clusters with replication support
# Usage: pgctl.sh <action> [options]
#        action: init|start|stop|status
################################################################################

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored messages
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Function to get total system memory in MB
get_total_memory_mb() {
    if command -v free >/dev/null 2>&1; then
        # Linux
        free -m | awk '/^Mem:/ {print $2}'
    elif [ -f /proc/meminfo ]; then
        # Linux via /proc/meminfo
        awk '/MemTotal/ {print int($2/1024)}' /proc/meminfo
    elif command -v sysctl >/dev/null 2>&1; then
        # macOS/BSD
        sysctl -n hw.memsize | awk '{print int($1/1024/1024)}'
    else
        # Default fallback: 8GB
        echo "8192"
    fi
}

# Function to calculate memory settings based on total memory
calculate_memory_settings() {
    local total_mem_mb=$1
    local max_mem_mb=$((total_mem_mb * 60 / 100))  # 60% of total memory

    # shared_buffers: typically 25% of available memory, but must be multiple of 128MB
    local shared_buffers_mb=$((max_mem_mb * 25 / 100))
    # Round down to nearest 128MB
    shared_buffers_mb=$((shared_buffers_mb / 128 * 128))
    # Minimum 128MB
    if [ $shared_buffers_mb -lt 128 ]; then
        shared_buffers_mb=128
    fi

    # maintenance_work_mem: typically 1GB for large systems, but adjust based on available memory
    local maintenance_work_mem_mb
    if [ $max_mem_mb -ge 8192 ]; then
        maintenance_work_mem_mb=1024
    elif [ $max_mem_mb -ge 4096 ]; then
        maintenance_work_mem_mb=512
    elif [ $max_mem_mb -ge 2048 ]; then
        maintenance_work_mem_mb=256
    else
        maintenance_work_mem_mb=128
    fi

    # max_wal_size: typically 4GB for large systems, but adjust based on available memory
    local max_wal_size_mb
    if [ $max_mem_mb -ge 16384 ]; then
        max_wal_size_mb=49152  # 48GB
    elif [ $max_mem_mb -ge 8192 ]; then
        max_wal_size_mb=24576  # 24GB
    elif [ $max_mem_mb -ge 4096 ]; then
        max_wal_size_mb=12288  # 12GB
    else
        max_wal_size_mb=4096   # 4GB
    fi

    # min_wal_size: typically 25% of max_wal_size
    local min_wal_size_mb=$((max_wal_size_mb / 4))

    echo "${shared_buffers_mb}MB|${maintenance_work_mem_mb}MB|${max_wal_size_mb}MB|${min_wal_size_mb}MB"
}

# Function to display usage
usage() {
    cat << EOF
Usage: $0 <action> [options]

Required:
    <action>            Action to perform: init|start|stop|status

Options:
    -c, --config FILE       Configuration file (default: ./pgctl.conf)
    --bin-dir DIR           PostgreSQL binary directory (default: use PATH)
                            If specified, uses DIR/initdb, DIR/pg_ctl, etc.
    --base-port PORT        Base port number (default: 5432)
    --wal-level LEVEL       WAL level: replica|logical (default: logical)
    --cluster-prefix PREFIX Cluster prefix (format: "1:s1,2:s2" or "s1,s2")
    --cluster-count COUNT   Number of clusters (default: 1)
    --node-count COUNT      Number of nodes per cluster: 2 or 3 (default: 3)
                            Note: 2 = 1 primary + 1 standby, 3 = 1 primary + 2 standbys
    --force                 (init only) Force reinitialization
    -h, --help              Show this help message

Examples:
    # Use default pgctl.conf in script directory
    $0 init
    $0 start

    # Specify custom config file
    $0 -c pgctl.conf init
    $0 init --config pgctl.conf

    # Override config file settings
    $0 init --base-port 6000 --wal-level logical --cluster-count 2 --node-count 3
    $0 init --cluster-prefix "s1,s2" --node-count 2

    # Specify PostgreSQL binary directory
    $0 init --bin-dir /usr/local/pgsql/bin
    $0 start --bin-dir /usr/local/pgsql/bin

    # Force reinitialization
    $0 init --force

Notes:
    - Data directory defaults to ./data/ (relative to current working directory)
    - Node naming: data/s1-1, data/s1-2, data/s1-3 (using hyphens)
    - Memory settings are automatically calculated based on system memory (60% max)
    - If --bin-dir is not specified, PostgreSQL binaries must be in PATH
    - Default values: BASE_PORT=5432, WAL_LEVEL=logical, CLUSTER_COUNT=1, NODE_COUNT=3
EOF
    exit 1
}

# Parse command line arguments
CONFIG_FILE=""
ACTION=""
FORCE_INIT=0
CLI_BASE_PORT=""
CLI_WAL_LEVEL=""
CLI_CLUSTER_PREFIX=""
CLI_CLUSTER_COUNT=""
CLI_NODE_COUNT=""
CLI_BIN_DIR=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        -c|--config)
            if [ -z "$2" ]; then
                log_error "Option $1 requires an argument"
                usage
            fi
            CONFIG_FILE="$2"
            shift 2
            ;;
        --base-port)
            if [ -z "$2" ]; then
                log_error "Option $1 requires an argument"
                usage
            fi
            CLI_BASE_PORT="$2"
            shift 2
            ;;
        --wal-level)
            if [ -z "$2" ]; then
                log_error "Option $1 requires an argument"
                usage
            fi
            if [[ "$2" != "replica" && "$2" != "logical" ]]; then
                log_error "Invalid wal-level: $2 (must be 'replica' or 'logical')"
                usage
            fi
            CLI_WAL_LEVEL="$2"
            shift 2
            ;;
        --cluster-prefix)
            if [ -z "$2" ]; then
                log_error "Option $1 requires an argument"
                usage
            fi
            CLI_CLUSTER_PREFIX="$2"
            shift 2
            ;;
        --bin-dir)
            if [ -z "$2" ]; then
                log_error "Option $1 requires an argument"
                usage
            fi
            CLI_BIN_DIR="$2"
            shift 2
            ;;
        --cluster-count)
            if [ -z "$2" ]; then
                log_error "Option $1 requires an argument"
                usage
            fi
            CLI_CLUSTER_COUNT="$2"
            shift 2
            ;;
        --node-count)
            if [ -z "$2" ]; then
                log_error "Option $1 requires an argument"
                usage
            fi
            if [[ "$2" != "2" && "$2" != "3" ]]; then
                log_error "Invalid node-count: $2 (must be 2 or 3)"
                usage
            fi
            CLI_NODE_COUNT="$2"
            shift 2
            ;;
        init|start|stop|status)
            if [ -n "$ACTION" ]; then
                log_error "Multiple actions specified: $ACTION and $1"
                usage
            fi
            ACTION="$1"
            shift
            ;;
        --force)
            FORCE_INIT=1
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            log_error "Unknown option or argument: $1"
            usage
            ;;
    esac
done

# Validate required arguments
if [ -z "$ACTION" ]; then
    log_error "Action not specified"
    usage
fi

# Determine config file
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
if [ -z "$CONFIG_FILE" ]; then
    CONFIG_FILE="$SCRIPT_DIR/pgctl.conf"
    if [ -f "$CONFIG_FILE" ]; then
        log_info "Using default config file: $CONFIG_FILE"
    else
        log_info "Config file not found, using built-in defaults"
        CONFIG_FILE=""
    fi
fi

# Source configuration file if exists
if [ -n "$CONFIG_FILE" ] && [ -f "$CONFIG_FILE" ]; then
    source "$CONFIG_FILE"
elif [ -n "$CONFIG_FILE" ]; then
    log_error "Config file not found: $CONFIG_FILE"
    exit 1
fi

# Set defaults for all parameters
# Data directory defaults to ./data/ relative to current working directory
BASE_DATA_DIR="${BASE_DATA_DIR:-$(pwd)/data}"
PG_BIN_DIR="${PG_BIN_DIR:-}"
NODE_COUNT="${NODE_COUNT:-3}"
CLUSTER_COUNT="${CLUSTER_COUNT:-1}"
WAL_LEVEL="${WAL_LEVEL:-logical}"

# Command line overrides config file
if [ -n "$CLI_BASE_PORT" ]; then
    BASE_PORT="$CLI_BASE_PORT"
else
    BASE_PORT="${BASE_PORT:-5432}"
fi

if [ -n "$CLI_WAL_LEVEL" ]; then
    WAL_LEVEL="$CLI_WAL_LEVEL"
fi

if [ -n "$CLI_CLUSTER_COUNT" ]; then
    CLUSTER_COUNT="$CLI_CLUSTER_COUNT"
fi

if [ -n "$CLI_NODE_COUNT" ]; then
    NODE_COUNT="$CLI_NODE_COUNT"
fi

# Command line overrides config file for bin directory
if [ -n "$CLI_BIN_DIR" ]; then
    PG_BIN_DIR="$CLI_BIN_DIR"
fi

# Validate node count
if [[ "$NODE_COUNT" != "2" && "$NODE_COUNT" != "3" ]]; then
    log_error "Invalid NODE_COUNT: $NODE_COUNT (must be 2 or 3)"
    exit 1
fi

# Parse cluster prefix from command line
if [ -n "$CLI_CLUSTER_PREFIX" ]; then
    # Parse format: "1:s1,2:s2" or "s1,s2"
    IFS=',' read -ra PREFIX_ARRAY <<< "$CLI_CLUSTER_PREFIX"
    for ((i=0; i<${#PREFIX_ARRAY[@]}; i++)); do
        prefix="${PREFIX_ARRAY[$i]}"
        # Check if format is "N:prefix" or just "prefix"
        if [[ "$prefix" =~ ^([0-9]+):(.+)$ ]]; then
            cluster_num="${BASH_REMATCH[1]}"
            cluster_prefix="${BASH_REMATCH[2]}"
            eval "CLUSTER_${cluster_num}_PREFIX=\"${cluster_prefix}\""
        else
            # Sequential assignment: first prefix goes to cluster 1, etc.
            cluster_num=$((i + 1))
            eval "CLUSTER_${cluster_num}_PREFIX=\"${prefix}\""
        fi
    done
fi

# Initialize cluster prefix array with defaults if not set
for ((i=1; i<=CLUSTER_COUNT; i++)); do
    prefix_var="CLUSTER_${i}_PREFIX"
    if [ -z "${!prefix_var}" ]; then
        eval "${prefix_var}=\"s${i}\""
    fi
done

# Set replication mode (backward compatibility)
REPLICATION_MODE="${REPLICATION_MODE:-async}"

# Validate --force is only used with init
if [ $FORCE_INIT -eq 1 ] && [ "$ACTION" != "init" ]; then
    log_warn "--force option is only applicable to 'init' action, ignoring..."
    FORCE_INIT=0
fi

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check if a file exists and is executable
file_exists_and_executable() {
    [ -f "$1" ] && [ -x "$1" ]
}

# Construct pg binary paths
if [ -n "$PG_BIN_DIR" ]; then
    # Use specified bin directory
    if [ ! -d "$PG_BIN_DIR" ]; then
        log_error "PostgreSQL binary directory does not exist: $PG_BIN_DIR"
        exit 1
    fi
    INITDB="$PG_BIN_DIR/initdb"
    PG_CTL="$PG_BIN_DIR/pg_ctl"
    PSQL="$PG_BIN_DIR/psql"
    PG_BASEBACKUP="$PG_BIN_DIR/pg_basebackup"

    # Verify binaries exist
    if ! file_exists_and_executable "$INITDB"; then
        log_error "initdb not found or not executable: $INITDB"
        exit 1
    fi
    if ! file_exists_and_executable "$PG_CTL"; then
        log_error "pg_ctl not found or not executable: $PG_CTL"
        exit 1
    fi
    if ! file_exists_and_executable "$PSQL"; then
        log_error "psql not found or not executable: $PSQL"
        exit 1
    fi
    if ! file_exists_and_executable "$PG_BASEBACKUP"; then
        log_error "pg_basebackup not found or not executable: $PG_BASEBACKUP"
        exit 1
    fi
    log_info "Using PostgreSQL binaries from: $PG_BIN_DIR"
else
    # Use PATH
    INITDB="initdb"
    PG_CTL="pg_ctl"
    PSQL="psql"
    PG_BASEBACKUP="pg_basebackup"

    # Verify binaries exist in PATH
    if ! command_exists "$INITDB"; then
        log_error "initdb not found in PATH. Please install PostgreSQL or use --bin-dir option"
        exit 1
    fi
    if ! command_exists "$PG_CTL"; then
        log_error "pg_ctl not found in PATH. Please install PostgreSQL or use --bin-dir option"
        exit 1
    fi
    if ! command_exists "$PSQL"; then
        log_error "psql not found in PATH. Please install PostgreSQL or use --bin-dir option"
        exit 1
    fi
    if ! command_exists "$PG_BASEBACKUP"; then
        log_error "pg_basebackup not found in PATH. Please install PostgreSQL or use --bin-dir option"
        exit 1
    fi
    log_info "Using PostgreSQL binaries from PATH"
fi

# Function to get port for a specific cluster and node
get_port() {
    local cluster_id=$1
    local node_id=$2
    echo $((BASE_PORT + (cluster_id - 1) * 100 + node_id - 1))
}

# Function to get cluster prefix
get_cluster_prefix() {
    local cluster_id=$1
    local prefix_var="CLUSTER_${cluster_id}_PREFIX"
    echo "${!prefix_var}"
}

# Function to get data directory for a specific cluster and node
get_data_dir() {
    local cluster_id=$1
    local node_id=$2
    local prefix=$(get_cluster_prefix $cluster_id)
    echo "$BASE_DATA_DIR/${prefix}-${node_id}"
}

# Function to get replication slot name
get_replication_slot_name() {
    local cluster_id=$1
    local node_id=$2
    echo "slot_c${cluster_id}_n${node_id}"
}

# Function to generate PostgreSQL configuration
generate_postgresql_conf() {
    local port=$1
    local is_primary=$2

    # Get memory settings
    local total_mem_mb=$(get_total_memory_mb)
    local mem_settings=$(calculate_memory_settings $total_mem_mb)
    IFS='|' read -ra MEM_VALUES <<< "$mem_settings"
    local shared_buffers="${MEM_VALUES[0]}"
    local maintenance_work_mem="${MEM_VALUES[1]}"
    local max_wal_size="${MEM_VALUES[2]}"
    local min_wal_size="${MEM_VALUES[3]}"

    # Log to stderr to avoid polluting the config file
    log_info "Memory settings: Total=${total_mem_mb}MB, SharedBuffers=${shared_buffers}, MaintenanceWorkMem=${maintenance_work_mem}" >&2

    cat << EOF
# PostgreSQL Configuration - Auto-generated
# Memory settings calculated from system memory (60% max)

listen_addresses = '0.0.0.0'
port = $port
max_connections = 5000
superuser_reserved_connections = 13
#unix_socket_directories = '.'
tcp_keepalives_idle = 60
tcp_keepalives_interval = 10
tcp_keepalives_count = 10
shared_buffers = $shared_buffers
huge_pages = try
work_mem = 4MB
maintenance_work_mem = $maintenance_work_mem
dynamic_shared_memory_type = posix
shared_preload_libraries = 'pg_stat_statements'
vacuum_cost_delay = 0
bgwriter_delay = 10ms
bgwriter_lru_maxpages = 1000
bgwriter_lru_multiplier = 5.0
effective_io_concurrency = 0
max_worker_processes = 128
wal_level = $WAL_LEVEL
#synchronous_commit = remote_write
synchronous_commit = remote_apply
full_page_writes = on
wal_buffers = 64MB
wal_writer_delay = 10ms
checkpoint_timeout = 30min
max_wal_size = $max_wal_size
min_wal_size = $min_wal_size
checkpoint_completion_target = 0.1
archive_mode = on
archive_command = '/bin/date'
max_wal_senders = 8
#wal_keep_segments = 4096
wal_sender_timeout = 15s
hot_standby = on
max_standby_archive_delay = 600s
max_standby_streaming_delay = 600s
wal_receiver_status_interval = 1s
hot_standby_feedback = off
wal_receiver_timeout = 30s
wal_retrieve_retry_interval = 5s
random_page_cost = 1.1
# Logging configuration - Enable automatic log rotation
log_destination = 'csvlog'
logging_collector = on
log_directory = 'log'
# Log filename with day-of-month pattern: postgresql-01.log to postgresql-31.log
# This ensures only last 31 days (1 month) of logs are kept due to automatic overwrite
# Log filename with date pattern for rotation: postgresql-YYYY-MM-DD.log
log_filename = 'postgresql-%d.log'
# log_filename = 'postgresql-%Y-%m-%d.log'
# Log file permissions (0600 = read/write for owner only)
log_file_mode = 0600
# Enable log rotation: truncate on rotation, rotate daily or when size limit reached
log_truncate_on_rotation = on
log_rotation_age = 1d
log_rotation_size = 100MB
# With log_filename='postgresql-%d.log' and log_truncate_on_rotation=on,
# PostgreSQL will automatically overwrite logs older than 31 days (one month)
log_checkpoints = on
log_connections = on
log_disconnections = on
log_error_verbosity = verbose
log_line_prefix = '%m [%p] [%l-1] user=%u,db=%d,app=%a,client=%h '
log_lock_waits = on
log_statement = 'ddl'
# Log slow queries (queries taking longer than 1 second)
log_min_duration_statement = 1000
log_timezone = 'PRC'
autovacuum = on
log_autovacuum_min_duration = 0
autovacuum_max_workers = 8
autovacuum_freeze_max_age = 950000000
autovacuum_multixact_freeze_max_age = 1100000000
autovacuum_vacuum_cost_delay = 0
datestyle = 'iso, mdy'
timezone = 'PRC'
lc_messages = 'en_US.utf8'
lc_monetary = 'en_US.utf8'
lc_numeric = 'en_US.utf8'
lc_time = 'en_US.utf8'
default_text_search_config = 'pg_catalog.english'
# wal_log_hints = on    # 如果你需要用pg_rewind修复WAL的时间线差异, 需要开启它, 但是开启它会导致写wal变多, 请斟酌
log_replication_commands = on
log_min_messages = INFO
EOF

    # Configure synchronous replication based on mode (only for primary)
    if [ $is_primary -eq 1 ]; then
        case "$REPLICATION_MODE" in
            sync)
                # Synchronous replication (强同步)
                local standby_names=""
                for ((i=2; i<=$NODE_COUNT; i++)); do
                    if [ -n "$standby_names" ]; then
                        standby_names="${standby_names},"
                    fi
                    standby_names="${standby_names}$(get_replication_slot_name $cluster_id $i)"
                done
                cat << EOF
synchronous_commit = on
synchronous_standby_names = '$standby_names'
EOF
                ;;
            quorum)
                # Quorum-based synchronous replication (半同步)
                local quorum_count=$(((NODE_COUNT - 1) / 2))
                local standby_names=""
                for ((i=2; i<=$NODE_COUNT; i++)); do
                    if [ -n "$standby_names" ]; then
                        standby_names="${standby_names},"
                    fi
                    standby_names="${standby_names}$(get_replication_slot_name $cluster_id $i)"
                done
                cat << EOF
synchronous_commit = on
synchronous_standby_names = '$quorum_count ($standby_names)'
EOF
                ;;
            async|*)
                # Asynchronous replication (异步)
                cat << EOF
synchronous_commit = remote_apply
EOF
                ;;
        esac
    fi
}

# Function to initialize a single node
init_node() {
    local cluster_id=$1
    local node_id=$2
    local data_dir=$(get_data_dir $cluster_id $node_id)
    local port=$(get_port $cluster_id $node_id)
    local is_primary=0
    [ $node_id -eq 1 ] && is_primary=1

    if [ -d "$data_dir" ]; then
        log_warn "Data directory already exists: $data_dir (skipping initialization)"
        return 0
    fi

    log_info "Initializing Cluster $cluster_id, Node $node_id at $data_dir (Port: $port)"

    # Create data directory parent if needed
    mkdir -p "$(dirname "$data_dir")"

    # Initialize database
    "$INITDB" -D "$data_dir" --encoding=UTF8 --locale=C

    # Generate and write postgresql.conf
    generate_postgresql_conf $port $is_primary > "$data_dir/postgresql.conf"

    # Configure pg_hba.conf for replication
    cat >> "$data_dir/pg_hba.conf" << EOF

# Replication configuration
host    replication     replicator      0.0.0.0/0               trust
host    all             all             0.0.0.0/0               trust
EOF

    log_info "Node initialized successfully: Cluster $cluster_id, Node $node_id (Port: $port)"
}

# Function to setup replication for standby nodes
setup_replication() {
    local cluster_id=$1
    local node_id=$2
    local data_dir=$(get_data_dir $cluster_id $node_id)
    local port=$(get_port $cluster_id $node_id)
    local primary_port=$(get_port $cluster_id 1)
    local slot_name=$(get_replication_slot_name $cluster_id $node_id)
    local is_primary=0

    if [ $node_id -eq 1 ]; then
        # Primary node, no replication setup needed
        return 0
    fi

    log_info "Setting up replication for Cluster $cluster_id, Node $node_id"

    # Remove existing data directory for standby
    if [ -d "$data_dir" ]; then
        log_info "Removing existing standby data directory: $data_dir"
        rm -rf "$data_dir"
    fi

    # Create replication slot on primary before pg_basebackup
    log_info "Creating replication slot: $slot_name on primary (port $primary_port)"
    "$PSQL" -h localhost -p $primary_port -U replicator postgres -c \
        "SELECT pg_create_physical_replication_slot('$slot_name');" 2>/dev/null || \
        log_warn "Replication slot may already exist"

    # Use pg_basebackup to clone from primary
    # -R option creates postgresql.auto.conf with primary_conninfo
    # -S option specifies replication slot name
    log_info "Cloning data from primary using pg_basebackup..."
    log_info "Command: $PG_BASEBACKUP -h localhost -p $primary_port -U replicator -D $data_dir -Fp -Xs -P -R -S $slot_name"
    if ! "$PG_BASEBACKUP" -h localhost -p $primary_port -U replicator -D "$data_dir" \
        -Fp -Xs -P -R -S "$slot_name"; then
        log_error "pg_basebackup failed for Cluster $cluster_id, Node $node_id"
        return 1
    fi

    # Verify pg_basebackup created the data directory
    if [ ! -d "$data_dir" ]; then
        log_error "Data directory was not created by pg_basebackup: $data_dir"
        return 1
    fi

    # Generate and update postgresql.conf with correct port and settings
    # Note: postgresql.auto.conf (created by -R) has higher priority and contains primary_conninfo
    log_info "Updating postgresql.conf for standby node..."
    generate_postgresql_conf $port $is_primary > "$data_dir/postgresql.conf"

    # Ensure primary_slot_name is set in postgresql.auto.conf (pg_basebackup -R should set this, but verify)
    if [ -f "$data_dir/postgresql.auto.conf" ]; then
        # Check if primary_slot_name is already in postgresql.auto.conf
        if ! grep -q "primary_slot_name" "$data_dir/postgresql.auto.conf" 2>/dev/null; then
            log_info "Adding primary_slot_name to postgresql.auto.conf"
            echo "primary_slot_name = '$slot_name'" >> "$data_dir/postgresql.auto.conf"
        fi
    else
        # If postgresql.auto.conf doesn't exist, create it with replication settings
        log_warn "postgresql.auto.conf not found, creating it manually"
        cat > "$data_dir/postgresql.auto.conf" << EOF
# Standby replication configuration (auto-generated)
primary_conninfo = 'host=localhost port=$primary_port user=replicator'
primary_slot_name = '$slot_name'
EOF
    fi

    # Ensure standby.signal exists (required for PG 12+)
    if [ ! -f "$data_dir/standby.signal" ]; then
        log_info "Creating standby.signal file"
        touch "$data_dir/standby.signal"
    fi

    # Verify replication configuration
    log_info "Verifying replication configuration..."
    if [ -f "$data_dir/standby.signal" ] && [ -f "$data_dir/postgresql.auto.conf" ]; then
        log_info "✓ Standby signal file exists"
        log_info "✓ Replication configuration file exists"
        if grep -q "primary_conninfo" "$data_dir/postgresql.auto.conf" 2>/dev/null; then
            log_info "✓ primary_conninfo configured"
        else
            log_warn "primary_conninfo not found in postgresql.auto.conf"
        fi
        if grep -q "primary_slot_name" "$data_dir/postgresql.auto.conf" 2>/dev/null; then
            log_info "✓ primary_slot_name configured: $slot_name"
        else
            log_warn "primary_slot_name not found in postgresql.auto.conf"
        fi
    else
        log_error "Replication configuration incomplete"
        return 1
    fi

    log_info "Replication setup complete for Cluster $cluster_id, Node $node_id"
    log_info "Standby node will connect to primary at localhost:$primary_port using slot: $slot_name"
}

# Function to initialize all clusters
action_init() {
    log_info "Starting cluster initialization..."
    log_info "Configuration: $CLUSTER_COUNT clusters, $NODE_COUNT nodes per cluster"
    log_info "Replication mode: $REPLICATION_MODE"
    log_info "Data directory: $BASE_DATA_DIR"

    # Handle --force option
    if [ $FORCE_INIT -eq 1 ]; then
        log_warn "Force initialization requested"

        # Stop all running clusters
        log_info "Stopping any running PostgreSQL instances..."
        for ((cluster=1; cluster<=CLUSTER_COUNT; cluster++)); do
            for ((node=1; node<=NODE_COUNT; node++)); do
                local data_dir=$(get_data_dir $cluster $node)
                if [ -d "$data_dir" ]; then
                    if "$PG_CTL" -D "$data_dir" status >/dev/null 2>&1; then
                        log_info "Stopping Cluster $cluster, Node $node"
                        "$PG_CTL" -D "$data_dir" stop -m fast 2>/dev/null || true
                        sleep 1
                    fi
                fi
            done
        done

        # Remove existing data directories
        log_warn "Removing existing data directories..."
        for ((cluster=1; cluster<=CLUSTER_COUNT; cluster++)); do
            for ((node=1; node<=NODE_COUNT; node++)); do
                local data_dir=$(get_data_dir $cluster $node)
                if [ -d "$data_dir" ]; then
                    log_info "Removing $data_dir"
                    rm -rf "$data_dir"
                fi
            done
        done

        log_info "Force cleanup completed"
    fi

    # Create base directory
    mkdir -p "$BASE_DATA_DIR"

    for ((cluster=1; cluster<=CLUSTER_COUNT; cluster++)); do
        local cluster_prefix=$(get_cluster_prefix $cluster)
        log_info "=== Initializing Cluster $cluster (Prefix: ${cluster_prefix}) ==="

        # Initialize primary node only (standby nodes will be created via pg_basebackup)
        log_info "Initializing primary node (Cluster $cluster, Node 1)"
        init_node $cluster 1

        # Start primary node first
        log_info "Starting primary node (Cluster $cluster, Node 1)"
        local primary_data_dir=$(get_data_dir $cluster 1)
        "$PG_CTL" -D "$primary_data_dir" -l "$primary_data_dir/logfile" start

        # Wait for primary to be ready
        log_info "Waiting for primary to be ready..."
        sleep 5

        # Check if primary is actually running
        local primary_port=$(get_port $cluster 1)
        local max_attempts=10
        local attempt=0
        while [ $attempt -lt $max_attempts ]; do
            if "$PSQL" -h localhost -p $primary_port postgres -c "SELECT 1;" >/dev/null 2>&1; then
                log_info "Primary node is ready"
                break
            fi
            attempt=$((attempt + 1))
            log_info "Waiting for primary to be ready... (attempt $attempt/$max_attempts)"
            sleep 2
        done

        if [ $attempt -eq $max_attempts ]; then
            log_error "Primary node failed to start. Check log: $primary_data_dir/logfile"
            return 1
        fi

        # Create replication user
        log_info "Creating replication user..."
        "$PSQL" -h localhost -p $primary_port postgres -c \
            "CREATE USER replicator WITH REPLICATION PASSWORD 'replicator';" 2>/dev/null || \
            log_warn "Replication user may already exist"

        # Setup standby nodes using pg_basebackup
        for ((node=2; node<=NODE_COUNT; node++)); do
            log_info "Setting up standby node (Cluster $cluster, Node $node)"
            if ! setup_replication $cluster $node; then
                log_error "Failed to setup replication for Cluster $cluster, Node $node"
                return 1
            fi
        done

        # Stop primary for now (will be started with 'start' action)
        log_info "Stopping primary node (will be started with 'start' action)"
        "$PG_CTL" -D "$primary_data_dir" stop
        sleep 2

        log_info "Cluster $cluster initialization complete"
    done

    log_info "All clusters initialized successfully!"
}

# Function to start a single node
start_node() {
    local cluster_id=$1
    local node_id=$2
    local data_dir=$(get_data_dir $cluster_id $node_id)
    local port=$(get_port $cluster_id $node_id)

    if [ ! -d "$data_dir" ]; then
        log_error "Data directory does not exist: $data_dir"
        return 1
    fi

    # Check if already running
    if "$PG_CTL" -D "$data_dir" status >/dev/null 2>&1; then
        log_warn "Cluster $cluster_id, Node $node_id is already running (Port: $port)"
        return 0
    fi

    log_info "Starting Cluster $cluster_id, Node $node_id (Port: $port)"
    "$PG_CTL" -D "$data_dir" -l "$data_dir/logfile" start

    # Wait a bit for startup
    sleep 1
}

# Function to start all clusters
action_start() {
    log_info "Starting all PostgreSQL clusters..."

    for ((cluster=1; cluster<=CLUSTER_COUNT; cluster++)); do
        log_info "=== Starting Cluster $cluster ==="

        # Start primary first
        start_node $cluster 1
        sleep 2

        # Start standby nodes
        for ((node=2; node<=NODE_COUNT; node++)); do
            start_node $cluster $node
            sleep 1
        done

        log_info "Cluster $cluster started"
    done

    log_info "All clusters started successfully!"
}

# Function to stop a single node
stop_node() {
    local cluster_id=$1
    local node_id=$2
    local data_dir=$(get_data_dir $cluster_id $node_id)
    local port=$(get_port $cluster_id $node_id)

    if [ ! -d "$data_dir" ]; then
        log_warn "Data directory does not exist: $data_dir"
        return 0
    fi

    # Check if running
    if ! "$PG_CTL" -D "$data_dir" status >/dev/null 2>&1; then
        log_warn "Cluster $cluster_id, Node $node_id is not running"
        return 0
    fi

    log_info "Stopping Cluster $cluster_id, Node $node_id (Port: $port)"
    "$PG_CTL" -D "$data_dir" stop -m fast
}

# Function to stop all clusters
action_stop() {
    log_info "Stopping all PostgreSQL clusters..."

    for ((cluster=1; cluster<=CLUSTER_COUNT; cluster++)); do
        log_info "=== Stopping Cluster $cluster ==="

        # Stop standby nodes first
        for ((node=$NODE_COUNT; node>=2; node--)); do
            stop_node $cluster $node
        done

        # Stop primary last
        stop_node $cluster 1

        log_info "Cluster $cluster stopped"
    done

    log_info "All clusters stopped successfully!"
}

# Function to check status of all clusters
action_status() {
    log_info "Checking status of all PostgreSQL clusters..."

    for ((cluster=1; cluster<=CLUSTER_COUNT; cluster++)); do
        echo ""
        local prefix=$(get_cluster_prefix $cluster)
        log_info "=== Cluster $cluster Status (Prefix: ${prefix}) ==="

        for ((node=1; node<=NODE_COUNT; node++)); do
            local data_dir=$(get_data_dir $cluster $node)
            local port=$(get_port $cluster $node)
            local role="Primary"
            [ $node -ne 1 ] && role="Standby"

            if [ ! -d "$data_dir" ]; then
                echo "  Node $node ($role, Port $port): NOT INITIALIZED"
                continue
            fi

            if "$PG_CTL" -D "$data_dir" status >/dev/null 2>&1; then
                echo -e "  Node $node ($role, Port $port, Data: $data_dir): ${GREEN}RUNNING${NC}"
            else
                echo -e "  Node $node ($role, Port $port, Data: $data_dir): ${RED}STOPPED${NC}"
            fi
        done
    done
    echo ""
}

# Main action dispatcher
case "$ACTION" in
    init)
        action_init
        ;;
    start)
        action_start
        ;;
    stop)
        action_stop
        ;;
    status)
        action_status
        ;;
    *)
        log_error "Unknown action: $ACTION"
        usage
        ;;
esac

exit 0
