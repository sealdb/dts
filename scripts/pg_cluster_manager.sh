#!/usr/bin/env bash

################################################################################
# PostgreSQL Cluster Manager Script
#
# Description: Manage multiple PostgreSQL clusters with replication support
# Usage: pg_cluster_manager.sh <config_file> <action>
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

# Function to display usage
usage() {
    cat << EOF
Usage: $0 <action> [options]

Required:
    <action>            Action to perform: init|start|stop|status

Options:
    -c, --config FILE       Configuration file (default: ./config.conf)
    --base-port PORT        Base port number (default: 5432)
    --wal-level LEVEL       WAL level: replica|logical (default: replica)
    --cluster-prefix PREFIX Cluster prefix (format: "1:s1_,2:s2_" or "s1_,s2_")
    --force                 (init only) Force reinitialization
    -h, --help              Show this help message

Examples:
    # Use default config.conf in script directory
    $0 init
    $0 start

    # Specify custom config file
    $0 -c config.conf init
    $0 init --config config.conf

    # Override config file settings
    $0 init --base-port 6000 --wal-level logical
    $0 -c config.conf init --cluster-prefix "s1_,s2_"
    $0 start --base-port 5432 --cluster-prefix "1:primary_,2:standby_"

    # Force reinitialization
    $0 init --force
    $0 --config config.conf init --force

Notes:
    - If -c/--config is not specified, looks for config.conf in script directory
    - If config file doesn't exist, uses built-in defaults
    - Command line options override config file settings
    - Default values: BASE_PORT=5432, WAL_LEVEL=replica, CLUSTER_PREFIX=s1_,s2_,...
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
if [ -z "$CONFIG_FILE" ]; then
    CONFIG_FILE="$SCRIPT_DIR/config.conf"
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
BASE_DATA_DIR="${BASE_DATA_DIR:-/tmp/pg_clusters}"
PG_BIN_DIR="${PG_BIN_DIR:-}"
NODE_COUNT="${NODE_COUNT:-3}"
CLUSTER_COUNT="${CLUSTER_COUNT:-2}"
WAL_LEVEL="${WAL_LEVEL:-replica}"

# Command line overrides config file
if [ -n "$CLI_BASE_PORT" ]; then
    BASE_PORT="$CLI_BASE_PORT"
else
    BASE_PORT="${BASE_PORT:-5432}"
fi

if [ -n "$CLI_WAL_LEVEL" ]; then
    WAL_LEVEL="$CLI_WAL_LEVEL"
fi

# Parse cluster prefix from command line
if [ -n "$CLI_CLUSTER_PREFIX" ]; then
    # Parse format: "1:s1_,2:s2_" or "s1_,s2_"
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
        eval "${prefix_var}=\"s${i}_\""
    fi
done

# Set replication mode (backward compatibility)
REPLICATION_MODE="${REPLICATION_MODE:-async}"

# Validate --force is only used with init
if [ $FORCE_INIT -eq 1 ] && [ "$ACTION" != "init" ]; then
    log_warn "--force option is only applicable to 'init' action, ignoring..."
    FORCE_INIT=0
fi

# Construct pg binary paths
if [ -n "$PG_BIN_DIR" ]; then
    INITDB="$PG_BIN_DIR/initdb"
    PG_CTL="$PG_BIN_DIR/pg_ctl"
    PSQL="$PG_BIN_DIR/psql"
    PG_BASEBACKUP="$PG_BIN_DIR/pg_basebackup"
else
    INITDB="initdb"
    PG_CTL="pg_ctl"
    PSQL="psql"
    PG_BASEBACKUP="pg_basebackup"
fi

# Function to get port for a specific cluster and node
get_port() {
    local cluster_id=$1
    local node_id=$2
    echo $((BASE_PORT + cluster_id * 100 + node_id))
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
    echo "$BASE_DATA_DIR/${prefix}${node_id}"
}

# Function to get replication slot name
get_replication_slot_name() {
    local cluster_id=$1
    local node_id=$2
    echo "slot_c${cluster_id}_n${node_id}"
}

# Function to initialize a single node
init_node() {
    local cluster_id=$1
    local node_id=$2
    local data_dir=$(get_data_dir $cluster_id $node_id)
    local port=$(get_port $cluster_id $node_id)

    if [ -d "$data_dir" ]; then
        log_warn "Data directory already exists: $data_dir (skipping initialization)"
        return 0
    fi

    log_info "Initializing Cluster $cluster_id, Node $node_id at $data_dir"

    # Create data directory parent if needed
    mkdir -p "$(dirname "$data_dir")"

    # Initialize database
    "$INITDB" -D "$data_dir" --encoding=UTF8 --locale=C

    # Configure postgresql.conf
    cat >> "$data_dir/postgresql.conf" << EOF

# Custom configuration
port = $port
max_connections = 100
shared_buffers = 128MB
wal_level = $WAL_LEVEL
max_wal_senders = 10
max_replication_slots = 10
hot_standby = on
listen_addresses = '*'
EOF

    # Configure synchronous replication based on mode
    if [ $node_id -eq 1 ]; then
        # Primary node configuration
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
                cat >> "$data_dir/postgresql.conf" << EOF
synchronous_commit = on
synchronous_standby_names = '$standby_names'
EOF
                log_info "Cluster $cluster_id configured for synchronous replication"
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
                cat >> "$data_dir/postgresql.conf" << EOF
synchronous_commit = on
synchronous_standby_names = '$quorum_count ($standby_names)'
EOF
                log_info "Cluster $cluster_id configured for quorum synchronous replication (quorum: $quorum_count)"
                ;;
            async|*)
                # Asynchronous replication (异步)
                cat >> "$data_dir/postgresql.conf" << EOF
synchronous_commit = off
EOF
                log_info "Cluster $cluster_id configured for asynchronous replication"
                ;;
        esac
    fi

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

    if [ $node_id -eq 1 ]; then
        # Primary node, no replication setup needed
        return 0
    fi

    log_info "Setting up replication for Cluster $cluster_id, Node $node_id"

    # Remove existing data directory for standby
    if [ -d "$data_dir" ]; then
        rm -rf "$data_dir"
    fi

    # Create replication slot on primary
    log_info "Creating replication slot: $slot_name on primary (port $primary_port)"
    "$PSQL" -h localhost -p $primary_port -U replicator postgres -c \
        "SELECT pg_create_physical_replication_slot('$slot_name');" 2>/dev/null || \
        log_warn "Replication slot may already exist"

    # Use pg_basebackup to clone from primary
    log_info "Cloning data from primary using pg_basebackup..."
    "$PG_BASEBACKUP" -h localhost -p $primary_port -U replicator -D "$data_dir" \
        -Fp -Xs -P -R -S "$slot_name"

    # Update standby configuration
    cat >> "$data_dir/postgresql.conf" << EOF

# Standby-specific configuration
port = $port
primary_slot_name = '$slot_name'
EOF

    # Create standby.signal if needed (for PG 12+)
    if [ ! -f "$data_dir/standby.signal" ]; then
        touch "$data_dir/standby.signal"
    fi

    log_info "Replication setup complete for Cluster $cluster_id, Node $node_id"
}

# Function to initialize all clusters
action_init() {
    log_info "Starting cluster initialization..."
    log_info "Configuration: $CLUSTER_COUNT clusters, $NODE_COUNT nodes per cluster"
    log_info "Replication mode: $REPLICATION_MODE"

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

        # Initialize all nodes
        for ((node=1; node<=NODE_COUNT; node++)); do
            init_node $cluster $node
        done

        # Start primary node first
        log_info "Starting primary node (Cluster $cluster, Node 1)"
        local primary_data_dir=$(get_data_dir $cluster 1)
        "$PG_CTL" -D "$primary_data_dir" -l "$primary_data_dir/logfile" start

        # Wait for primary to be ready
        sleep 3

        # Create replication user
        local primary_port=$(get_port $cluster 1)
        "$PSQL" -h localhost -p $primary_port postgres -c \
            "CREATE USER replicator WITH REPLICATION PASSWORD 'replicator';" 2>/dev/null || \
            log_warn "Replication user may already exist"

        # Setup standby nodes
        for ((node=2; node<=NODE_COUNT; node++)); do
            setup_replication $cluster $node
        done

        # Stop primary for now (will be started with 'start' action)
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


