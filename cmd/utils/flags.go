// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// Package utils contains internal helper functions for go-ethereum commands.
package utils

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AlayaNetwork/Alaya-Go/miner"

	"github.com/AlayaNetwork/Alaya-Go/core/snapshotdb"

	"gopkg.in/urfave/cli.v1"

	"github.com/AlayaNetwork/Alaya-Go/accounts"
	"github.com/AlayaNetwork/Alaya-Go/accounts/keystore"
	"github.com/AlayaNetwork/Alaya-Go/common"
	"github.com/AlayaNetwork/Alaya-Go/common/fdlimit"
	"github.com/AlayaNetwork/Alaya-Go/consensus"
	"github.com/AlayaNetwork/Alaya-Go/consensus/cbft/types"
	"github.com/AlayaNetwork/Alaya-Go/core"
	"github.com/AlayaNetwork/Alaya-Go/core/vm"
	"github.com/AlayaNetwork/Alaya-Go/crypto"
	"github.com/AlayaNetwork/Alaya-Go/crypto/bls"
	"github.com/AlayaNetwork/Alaya-Go/eth"
	"github.com/AlayaNetwork/Alaya-Go/eth/downloader"
	"github.com/AlayaNetwork/Alaya-Go/eth/gasprice"
	"github.com/AlayaNetwork/Alaya-Go/ethdb"
	"github.com/AlayaNetwork/Alaya-Go/ethstats"
	"github.com/AlayaNetwork/Alaya-Go/les"
	"github.com/AlayaNetwork/Alaya-Go/log"
	"github.com/AlayaNetwork/Alaya-Go/metrics"
	"github.com/AlayaNetwork/Alaya-Go/metrics/influxdb"
	"github.com/AlayaNetwork/Alaya-Go/node"
	"github.com/AlayaNetwork/Alaya-Go/p2p"
	"github.com/AlayaNetwork/Alaya-Go/p2p/discover"
	"github.com/AlayaNetwork/Alaya-Go/p2p/nat"
	"github.com/AlayaNetwork/Alaya-Go/p2p/netutil"
	"github.com/AlayaNetwork/Alaya-Go/params"
)

var (
	CommandHelpTemplate = `{{.cmd.Name}}{{if .cmd.Subcommands}} command{{end}}{{if .cmd.Flags}} [command options]{{end}} [arguments...]
{{if .cmd.Description}}{{.cmd.Description}}
{{end}}{{if .cmd.Subcommands}}
SUBCOMMANDS:
	{{range .cmd.Subcommands}}{{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}
	{{end}}{{end}}{{if .categorizedFlags}}
{{range $idx, $categorized := .categorizedFlags}}{{$categorized.Name}} OPTIONS:
{{range $categorized.Flags}}{{"\t"}}{{.}}
{{end}}
{{end}}{{end}}`
	OriginCommandHelpTemplate = ""
)

func init() {
	OriginCommandHelpTemplate = cli.CommandHelpTemplate
	cli.AppHelpTemplate = `{{.Name}} {{if .Flags}}[global options] {{end}}command{{if .Flags}} [command options]{{end}} [arguments...]

VERSION:
   {{.Version}}

COMMANDS:
   {{range .Commands}}{{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}
   {{end}}{{if .Flags}}
GLOBAL OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{end}}
`

	cli.CommandHelpTemplate = CommandHelpTemplate
}

// NewApp creates an app with sane defaults.
func NewApp(gitCommit, gitDate, usage string) *cli.App {
	app := cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Author = ""
	//app.Authors = nil
	app.Email = ""
	app.Version = params.VersionWithCommit(gitCommit, gitDate)
	app.Usage = usage
	return app
}

// These are all the command line flags we support.
// If you add to this list, please remember to include the
// flag in the appropriate command definition.
//
// The flags are defined here so their names and help texts
// are the same for all commands.

var (
	// General settings
	DataDirFlag = DirectoryFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases and keystore",
		Value: DirectoryString{node.DefaultDataDir()},
	}
	AncientFlag = DirectoryFlag{
		Name:  "datadir.ancient",
		Usage: "Data directory for ancient chain segments (default = inside chaindata)",
	}
	KeyStoreDirFlag = DirectoryFlag{
		Name:  "keystore",
		Usage: "Directory for the keystore (default = inside the datadir)",
	}
	NoUSBFlag = cli.BoolFlag{
		Name:  "nousb",
		Usage: "Disables monitoring for and managing USB hardware wallets",
	}
	NetworkIdFlag = cli.Uint64Flag{
		Name:  "networkid",
		Usage: "Network identifier (integer, 1=Frontier, 2=Morden (disused), 3=Ropsten, 4=Rinkeby)",
		Value: eth.DefaultConfig.NetworkId,
	}
	AlayaNetFlag = cli.BoolFlag{
		Name:  "alaya",
		Usage: "alaya network: pre-configured alaya network",
	}
	AddressHRPFlag = cli.StringFlag{
		Name:  "addressHRP",
		Usage: "set the address hrp,if not set,use default address hrp",
	}
	DeveloperPeriodFlag = cli.IntFlag{
		Name:  "dev.period",
		Usage: "Block period to use in developer mode (0 = mine only if transaction pending)",
	}
	IdentityFlag = cli.StringFlag{
		Name:  "identity",
		Usage: "Custom node name",
	}
	DocRootFlag = DirectoryFlag{
		Name:  "docroot",
		Usage: "Document Root for HTTPClient file scheme",
		Value: DirectoryString{homeDir()},
	}
	defaultSyncMode = eth.DefaultConfig.SyncMode
	SyncModeFlag    = TextMarshalerFlag{
		Name:  "syncmode",
		Usage: `Blockchain sync mode ("fast", "full", or "light")`,
		Value: &defaultSyncMode,
	}
	LightServFlag = cli.IntFlag{
		Name:  "lightserv",
		Usage: "Maximum percentage of time allowed for serving LES requests (0-90)",
		Value: 0,
	}
	LightPeersFlag = cli.IntFlag{
		Name:  "lightpeers",
		Usage: "Maximum number of LES client peers",
		Value: eth.DefaultConfig.LightPeers,
	}
	LightKDFFlag = cli.BoolFlag{
		Name:  "lightkdf",
		Usage: "Reduce key-derivation RAM & CPU usage at some expense of KDF strength",
	}
	// Transaction pool settings
	TxPoolLocalsFlag = cli.StringFlag{
		Name:  "txpool.locals",
		Usage: "Comma separated accounts to treat as locals (no flush, priority inclusion)",
	}
	TxPoolNoLocalsFlag = cli.BoolFlag{
		Name:  "txpool.nolocals",
		Usage: "Disables price exemptions for locally submitted transactions",
	}
	TxPoolJournalFlag = cli.StringFlag{
		Name:  "txpool.journal",
		Usage: "Disk journal for local transaction to survive node restarts",
		Value: core.DefaultTxPoolConfig.Journal,
	}
	TxPoolRejournalFlag = cli.DurationFlag{
		Name:  "txpool.rejournal",
		Usage: "Time interval to regenerate the local transaction journal",
		Value: core.DefaultTxPoolConfig.Rejournal,
	}
	TxPoolPriceBumpFlag = cli.Uint64Flag{
		Name:  "txpool.pricebump",
		Usage: "Price bump percentage to replace an already existing transaction",
		Value: eth.DefaultConfig.TxPool.PriceBump,
	}
	TxPoolAccountSlotsFlag = cli.Uint64Flag{
		Name:  "txpool.accountslots",
		Usage: "Minimum number of executable transaction slots guaranteed per account",
		Value: eth.DefaultConfig.TxPool.AccountSlots,
	}
	TxPoolGlobalSlotsFlag = cli.Uint64Flag{
		Name:  "txpool.globalslots",
		Usage: "Maximum number of executable transaction slots for all accounts",
		Value: eth.DefaultConfig.TxPool.GlobalSlots,
	}
	TxPoolAccountQueueFlag = cli.Uint64Flag{
		Name:  "txpool.accountqueue",
		Usage: "Maximum number of non-executable transaction slots permitted per account",
		Value: eth.DefaultConfig.TxPool.AccountQueue,
	}
	TxPoolGlobalQueueFlag = cli.Uint64Flag{
		Name:  "txpool.globalqueue",
		Usage: "Maximum number of non-executable transaction slots for all accounts",
		Value: eth.DefaultConfig.TxPool.GlobalQueue,
	}
	TxPoolGlobalTxCountFlag = cli.Uint64Flag{
		Name:  "txpool.globaltxcount",
		Usage: "Maximum number of transactions for package",
		Value: eth.DefaultConfig.TxPool.GlobalTxCount,
	}
	TxPoolLifetimeFlag = cli.DurationFlag{
		Name:  "txpool.lifetime",
		Usage: "Maximum amount of time non-executable transaction are queued",
		Value: eth.DefaultConfig.TxPool.Lifetime,
	}
	TxPoolCacheSizeFlag = cli.Uint64Flag{
		Name:  "txpool.cacheSize",
		Usage: "After receiving the specified number of transactions from the remote, move the transactions in the queen to pending",
		Value: eth.DefaultConfig.TxPool.TxCacheSize,
	}
	// Performance tuning settings
	CacheFlag = cli.IntFlag{
		Name:  "cache",
		Usage: "Megabytes of memory allocated to internal caching",
		Value: 1024,
	}
	CacheDatabaseFlag = cli.IntFlag{
		Name:  "cache.database",
		Usage: "Percentage of cache memory allowance to use for database io",
		Value: 75,
	}
	CacheGCFlag = cli.IntFlag{
		Name:  "cache.gc",
		Usage: "Percentage of cache memory allowance to use for trie pruning (default = 25% full mode, 0% archive mode)",
		Value: 25,
	}
	CacheTrieDBFlag = cli.IntFlag{
		Name:  "cache.triedb",
		Usage: "Megabytes of memory allocated to triedb internal caching",
		Value: eth.DefaultConfig.TrieDBCache,
	}
	MinerGasTargetFlag = cli.Uint64Flag{
		Name:  "miner.gastarget",
		Usage: "Target gas floor for mined blocks",
		Value: eth.DefaultConfig.Miner.GasFloor,
	}
	MinerGasPriceFlag = BigFlag{
		Name:  "miner.gasprice",
		Usage: "Minimum gas price for mining a transaction",
		Value: eth.DefaultConfig.Miner.GasPrice,
	}
	// Account settings
	UnlockedAccountFlag = cli.StringFlag{
		Name:  "unlock",
		Usage: "Comma separated list of accounts to unlock",
		Value: "",
	}
	PasswordFileFlag = cli.StringFlag{
		Name:  "password",
		Usage: "Password file to use for non-interactive password input",
		Value: "",
	}
	InsecureUnlockAllowedFlag = cli.BoolFlag{
		Name:  "allow-insecure-unlock",
		Usage: "Allow insecure account unlocking when account-related RPCs are exposed by http",
	}
	RPCGlobalGasCap = cli.Uint64Flag{
		Name:  "rpc.gascap",
		Usage: "Sets a cap on gas that can be used in eth_call/estimateGas",
	}
	// Logging and debug settings
	EthStatsURLFlag = cli.StringFlag{
		Name:  "ethstats",
		Usage: "Reporting URL of a ethstats service (nodename:secret@host:port)",
	}
	NoCompactionFlag = cli.BoolFlag{
		Name:  "nocompaction",
		Usage: "Disables db compaction after import",
	}
	// RPC settings
	RPCEnabledFlag = cli.BoolFlag{
		Name:  "rpc",
		Usage: "Enable the HTTP-RPC server",
	}
	RPCListenAddrFlag = cli.StringFlag{
		Name:  "rpcaddr",
		Usage: "HTTP-RPC server listening interface",
		Value: node.DefaultHTTPHost,
	}
	RPCPortFlag = cli.IntFlag{
		Name:  "rpcport",
		Usage: "HTTP-RPC server listening port",
		Value: node.DefaultHTTPPort,
	}
	RPCCORSDomainFlag = cli.StringFlag{
		Name:  "rpccorsdomain",
		Usage: "Comma separated list of domains from which to accept cross origin requests (browser enforced)",
		Value: "",
	}
	RPCVirtualHostsFlag = cli.StringFlag{
		Name:  "rpcvhosts",
		Usage: "Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard.",
		Value: strings.Join(node.DefaultConfig.HTTPVirtualHosts, ","),
	}
	RPCApiFlag = cli.StringFlag{
		Name:  "rpcapi",
		Usage: "API's offered over the HTTP-RPC interface",
		Value: "",
	}
	IPCDisabledFlag = cli.BoolFlag{
		Name:  "ipcdisable",
		Usage: "Disable the IPC-RPC server",
	}
	IPCPathFlag = DirectoryFlag{
		Name:  "ipcpath",
		Usage: "Filename for IPC socket/pipe within the datadir (explicit paths escape it)",
	}
	WSEnabledFlag = cli.BoolFlag{
		Name:  "ws",
		Usage: "Enable the WS-RPC server",
	}
	WSListenAddrFlag = cli.StringFlag{
		Name:  "wsaddr",
		Usage: "WS-RPC server listening interface",
		Value: node.DefaultWSHost,
	}
	WSPortFlag = cli.IntFlag{
		Name:  "wsport",
		Usage: "WS-RPC server listening port",
		Value: node.DefaultWSPort,
	}
	WSApiFlag = cli.StringFlag{
		Name:  "wsapi",
		Usage: "API's offered over the WS-RPC interface",
		Value: "",
	}
	WSAllowedOriginsFlag = cli.StringFlag{
		Name:  "wsorigins",
		Usage: "Origins from which to accept websockets requests",
		Value: "",
	}
	ExecFlag = cli.StringFlag{
		Name:  "exec",
		Usage: "Execute JavaScript statement",
	}
	PreloadJSFlag = cli.StringFlag{
		Name:  "preload",
		Usage: "Comma separated list of JavaScript files to preload into the console",
	}

	// Network Settings
	MaxPeersFlag = cli.IntFlag{
		Name:  "maxpeers",
		Usage: "Maximum number of network peers (network disabled if set to 0)",
		Value: 60,
	}
	MaxConsensusPeersFlag = cli.IntFlag{
		Name:  "maxconsensuspeers",
		Usage: "Maximum number of network consensus peers (network disabled if set to 0)",
		Value: 40,
	}
	MaxPendingPeersFlag = cli.IntFlag{
		Name:  "maxpendpeers",
		Usage: "Maximum number of pending connection attempts (defaults used if set to 0)",
		Value: 0,
	}
	ListenPortFlag = cli.IntFlag{
		Name:  "port",
		Usage: "Network listening port",
		Value: 16789,
	}
	BootnodesFlag = cli.StringFlag{
		Name:  "bootnodes",
		Usage: "Comma separated enode URLs for P2P discovery bootstrap (set v4+v5 instead for light servers)",
		Value: "",
	}
	BootnodesV4Flag = cli.StringFlag{
		Name:  "bootnodesv4",
		Usage: "Comma separated enode URLs for P2P v4 discovery bootstrap (light server, full nodes)",
		Value: "",
	}
	/*
		BootnodesV5Flag = cli.StringFlag{
			Name:  "bootnodesv5",
			Usage: "Comma separated enode URLs for P2P v5 discovery bootstrap (light server, light nodes)",
			Value: "",
		}*/
	NodeKeyFileFlag = cli.StringFlag{
		Name:  "nodekey",
		Usage: "P2P node key file",
	}
	NodeKeyHexFlag = cli.StringFlag{
		Name:  "nodekeyhex",
		Usage: "P2P node key as hex (for testing)",
	}
	NATFlag = cli.StringFlag{
		Name:  "nat",
		Usage: "NAT port mapping mechanism (any|none|upnp|pmp|extip:<IP>)",
		Value: "any",
	}
	NoDiscoverFlag = cli.BoolFlag{
		Name:  "nodiscover",
		Usage: "Disables the peer discovery mechanism (manual peer addition)",
	}
	/*
		DiscoveryV5Flag = cli.BoolFlag{
			Name:  "v5disc",
			Usage: "Enables the experimental RLPx V5 (Topic Discovery) mechanism",
		}*/
	NetrestrictFlag = cli.StringFlag{
		Name:  "netrestrict",
		Usage: "Restricts network communication to the given IP networks (CIDR masks)",
	}

	// ATM the url is left to the user and deployment to
	JSpathFlag = cli.StringFlag{
		Name:  "jspath",
		Usage: "JavaScript root path for `loadScript`",
		Value: ".",
	}

	// Gas price oracle settings
	GpoBlocksFlag = cli.IntFlag{
		Name:  "gpoblocks",
		Usage: "Number of recent blocks to check for gas prices",
		Value: eth.DefaultConfig.GPO.Blocks,
	}
	GpoPercentileFlag = cli.IntFlag{
		Name:  "gpopercentile",
		Usage: "Suggested gas price is the given percentile of a set of recent transaction gas prices",
		Value: eth.DefaultConfig.GPO.Percentile,
	}
	//WhisperEnabledFlag = cli.BoolFlag{
	//	Name:  "shh",
	//	Usage: "Enable Whisper",
	//}
	//WhisperMaxMessageSizeFlag = cli.IntFlag{
	//	Name:  "shh.maxmessagesize",
	//	Usage: "Max message size accepted",
	//	Value: int(whisper.DefaultMaxMessageSize),
	//}
	//WhisperRestrictConnectionBetweenLightClientsFlag = cli.BoolFlag{
	//	Name:  "shh.restrict-light",
	//	Usage: "Restrict connection between two whisper light clients",
	//}

	// Metrics flags
	MetricsEnabledFlag = cli.BoolFlag{
		Name:  "metrics",
		Usage: "Enable metrics collection and reporting",
	}
	MetricsEnabledExpensiveFlag = cli.BoolFlag{
		Name:  "metrics.expensive",
		Usage: "Enable expensive metrics collection and reporting",
	}
	MetricsEnableInfluxDBFlag = cli.BoolFlag{
		Name:  "metrics.influxdb",
		Usage: "Enable metrics export/push to an external InfluxDB database",
	}
	MetricsInfluxDBEndpointFlag = cli.StringFlag{
		Name:  "metrics.influxdb.endpoint",
		Usage: "InfluxDB API endpoint to report metrics to",
		Value: "http://localhost:8086",
	}
	MetricsInfluxDBDatabaseFlag = cli.StringFlag{
		Name:  "metrics.influxdb.database",
		Usage: "InfluxDB database name to push reported metrics to",
		Value: "alaya",
	}
	MetricsInfluxDBUsernameFlag = cli.StringFlag{
		Name:  "metrics.influxdb.username",
		Usage: "Username to authorize access to the database",
		Value: "test",
	}
	MetricsInfluxDBPasswordFlag = cli.StringFlag{
		Name:  "metrics.influxdb.password",
		Usage: "Password to authorize access to the database",
		Value: "test",
	}
	// Tags are part of every measurement sent to InfluxDB. Queries on tags are faster in InfluxDB.
	// For example `host` tag could be used so that we can group all nodes and average a measurement
	// across all of them, but also so that we can select a specific node and inspect its measurements.
	// https://docs.influxdata.com/influxdb/v1.4/concepts/key_concepts/#tag-key
	MetricsInfluxDBTagsFlag = cli.StringFlag{
		Name:  "metrics.influxdb.tags",
		Usage: "Comma-separated InfluxDB tags (key/values) attached to all measurements",
		Value: "host=localhost",
	}

	// mpc compute
	//MPCIceFileFlag = cli.StringFlag{
	//	Name:  "mpc.ice",
	//	Usage: "Filename for ice to init mvm",
	//	Value: "",
	//}
	//MPCActorFlag = cli.StringFlag{
	//	Name:  "mpc.actor",
	//	Usage: "The address of actor to exec mpc compute",
	//	Value: "",
	//}
	//MPCEnabledFlag = cli.BoolFlag{
	//	Name:  "mpc",
	//	Usage: "Enable mpc compute",
	//}
	//VCEnabledFlag = cli.BoolFlag{
	//	Name:  "vc",
	//	Usage: "Enable vc compute",
	//}
	//VCActorFlag = cli.StringFlag{
	//	Name:  "vc.actor",
	//	Usage: "The address of vc to exec set result",
	//	Value: "",
	//}
	//
	//VCPasswordFlag = cli.StringFlag{
	//	Name:  "vc.password",
	//	Usage: "the pwd of unlock actor",
	//	Value: "",
	//}

	CbftPeerMsgQueueSize = cli.Uint64Flag{
		Name:  "cbft.msg_queue_size",
		Usage: "Message queue size",
		Value: 1024,
	}

	CbftWalDisabledFlag = cli.BoolFlag{
		Name:  "cbft.wal.disabled",
		Usage: "Disable the Wal server",
	}

	CbftMaxPingLatency = cli.Int64Flag{
		Name:  "cbft.max_ping_latency",
		Usage: "Maximum latency of ping",
		Value: 2000,
	}

	CbftBlsPriKeyFileFlag = cli.StringFlag{
		Name:  "cbft.blskey",
		Usage: "BLS key file",
	}

	CbftBlacklistDeadlineFlag = cli.StringFlag{
		Name:  "cbft.blacklist_deadline",
		Usage: "Blacklist effective time. uint:minute",
		Value: "60",
	}

	DBNoGCFlag = cli.BoolFlag{
		Name:  "db.nogc",
		Usage: "Disables database garbage collection",
	}
	DBGCIntervalFlag = cli.Uint64Flag{
		Name:  "db.gc_interval",
		Usage: "Block interval for garbage collection",
		Value: eth.DefaultConfig.DBGCInterval,
	}
	DBGCTimeoutFlag = cli.DurationFlag{
		Name:  "db.gc_timeout",
		Usage: "Maximum time for database garbage collection",
		Value: eth.DefaultConfig.DBGCTimeout,
	}
	DBGCMptFlag = cli.BoolFlag{
		Name:  "db.gc_mpt",
		Usage: "Enables database garbage collection MPT",
	}
	DBGCBlockFlag = cli.IntFlag{
		Name:  "db.gc_block",
		Usage: "Number of cache block states, default 10",
		Value: eth.DefaultConfig.DBGCBlock,
	}

	VMWasmType = cli.StringFlag{
		Name:   "vm.wasm_type",
		Usage:  "The actual implementation type of the wasm instance",
		EnvVar: "",
		Value:  eth.DefaultConfig.VMWasmType,
	}

	VmTimeoutDuration = cli.Uint64Flag{
		Name:   "vm.timeout_duration",
		Usage:  "The VM execution timeout duration (uint: ms)",
		EnvVar: "",
		Value:  eth.DefaultConfig.VmTimeoutDuration,
	}
)

// MakeDataDir retrieves the currently requested data directory, terminating
// if none (or the empty string) is specified. If the node is starting a testnet,
// the a subdirectory of the specified datadir will be used.
func MakeDataDir(ctx *cli.Context) string {
	if path := ctx.GlobalString(DataDirFlag.Name); path != "" {
		return filepath.Join(path, "alayanet")
	}
	Fatalf("Cannot determine default data directory, please set manually (--datadir)")
	return ""
}

// setNodeKey creates a node key from set command line flags, either loading it
// from a file or as a specified hex value. If neither flags were provided, this
// method returns nil and an emphemeral key is to be generated.
func setNodeKey(ctx *cli.Context, cfg *p2p.Config) {
	var (
		hex  = ctx.GlobalString(NodeKeyHexFlag.Name)
		file = ctx.GlobalString(NodeKeyFileFlag.Name)
		key  *ecdsa.PrivateKey
		err  error
	)
	switch {
	case file != "" && hex != "":
		Fatalf("Options %q and %q are mutually exclusive", NodeKeyFileFlag.Name, NodeKeyHexFlag.Name)
	case file != "":
		if key, err = crypto.LoadECDSA(file); err != nil {
			Fatalf("Option %q: %v", NodeKeyFileFlag.Name, err)
		}
		cfg.PrivateKey = key
	case hex != "":
		if key, err = crypto.HexToECDSA(hex); err != nil {
			Fatalf("Option %q: %v", NodeKeyHexFlag.Name, err)
		}
		cfg.PrivateKey = key
	}
}

// setNodeUserIdent creates the user identifier from CLI flags.
func setNodeUserIdent(ctx *cli.Context, cfg *node.Config) {
	if identity := ctx.GlobalString(IdentityFlag.Name); len(identity) > 0 {
		cfg.UserIdent = identity
	}
}

// setBootstrapNodes creates a list of bootstrap nodes from the command line
// flags, reverting to pre-configured ones if none have been specified.
func setBootstrapNodes(ctx *cli.Context, cfg *p2p.Config) {
	urls := params.AlayanetBootnodes

	switch {
	case ctx.GlobalIsSet(BootnodesFlag.Name) || ctx.GlobalIsSet(BootnodesV4Flag.Name):
		if ctx.GlobalIsSet(BootnodesV4Flag.Name) {
			urls = strings.Split(ctx.GlobalString(BootnodesV4Flag.Name), ",")
		} else {
			urls = strings.Split(ctx.GlobalString(BootnodesFlag.Name), ",")
		}
	case ctx.GlobalBool(AlayaNetFlag.Name):
		urls = params.AlayanetBootnodes
	case cfg.BootstrapNodes != nil:
		return // already set, don't apply defaults.
	}

	cfg.BootstrapNodes = make([]*discover.Node, 0, len(urls))
	for _, url := range urls {
		if url != "" {
			node, err := discover.ParseNode(url)
			if err != nil {
				log.Crit("Bootstrap URL invalid", "enode", url, "err", err)
			}
			cfg.BootstrapNodes = append(cfg.BootstrapNodes, node)
		}
	}
}

// setBootstrapNodesV5 creates a list of bootstrap nodes from the command line
// flags, reverting to pre-configured ones if none have been specified.
/*
func setBootstrapNodesV5(ctx *cli.Context, cfg *p2p.Config) {
	urls := params.DiscoveryV5Bootnodes
	switch {
	case ctx.GlobalIsSet(BootnodesFlag.Name) || ctx.GlobalIsSet(BootnodesV5Flag.Name):
		if ctx.GlobalIsSet(BootnodesV5Flag.Name) {
			urls = strings.Split(ctx.GlobalString(BootnodesV5Flag.Name), ",")
		} else {
			urls = strings.Split(ctx.GlobalString(BootnodesFlag.Name), ",")
		}
	case cfg.BootstrapNodesV5 != nil:
		return // already set, don't apply defaults.
	}

	cfg.BootstrapNodesV5 = make([]*discv5.Node, 0, len(urls))
	for _, url := range urls {
		node, err := discv5.ParseNode(url)
		if err != nil {
			log.Error("Bootstrap URL invalid", "enode", url, "err", err)
			continue
		}
		cfg.BootstrapNodesV5 = append(cfg.BootstrapNodesV5, node)
	}
}*/

// setListenAddress creates a TCP listening address string from set command
// line flags.
func setListenAddress(ctx *cli.Context, cfg *p2p.Config) {
	if ctx.GlobalIsSet(ListenPortFlag.Name) {
		cfg.ListenAddr = fmt.Sprintf(":%d", ctx.GlobalInt(ListenPortFlag.Name))
	}
}

// setNAT creates a port mapper from command line flags.
func setNAT(ctx *cli.Context, cfg *p2p.Config) {
	if ctx.GlobalIsSet(NATFlag.Name) {
		natif, err := nat.Parse(ctx.GlobalString(NATFlag.Name))
		if err != nil {
			Fatalf("Option %s: %v", NATFlag.Name, err)
		}
		cfg.NAT = natif
	}
}

// splitAndTrim splits input separated by a comma
// and trims excessive white space from the substrings.
func splitAndTrim(input string) []string {
	result := strings.Split(input, ",")
	for i, r := range result {
		result[i] = strings.TrimSpace(r)
	}
	return result
}

// setHTTP creates the HTTP RPC listener interface string from the set
// command line flags, returning empty if the HTTP endpoint is disabled.
func setHTTP(ctx *cli.Context, cfg *node.Config) {
	if ctx.GlobalBool(RPCEnabledFlag.Name) && cfg.HTTPHost == "" {
		cfg.HTTPHost = "127.0.0.1"
		if ctx.GlobalIsSet(RPCListenAddrFlag.Name) {
			cfg.HTTPHost = ctx.GlobalString(RPCListenAddrFlag.Name)
		}
	}

	if ctx.GlobalIsSet(RPCPortFlag.Name) {
		cfg.HTTPPort = ctx.GlobalInt(RPCPortFlag.Name)
	}
	if ctx.GlobalIsSet(RPCCORSDomainFlag.Name) {
		cfg.HTTPCors = splitAndTrim(ctx.GlobalString(RPCCORSDomainFlag.Name))
	}
	if ctx.GlobalIsSet(RPCApiFlag.Name) {
		cfg.HTTPModules = splitAndTrim(ctx.GlobalString(RPCApiFlag.Name))
	}
	if ctx.GlobalIsSet(RPCVirtualHostsFlag.Name) {
		cfg.HTTPVirtualHosts = splitAndTrim(ctx.GlobalString(RPCVirtualHostsFlag.Name))
	}
}

// setWS creates the WebSocket RPC listener interface string from the set
// command line flags, returning empty if the HTTP endpoint is disabled.
func setWS(ctx *cli.Context, cfg *node.Config) {
	if ctx.GlobalBool(WSEnabledFlag.Name) && cfg.WSHost == "" {
		cfg.WSHost = "127.0.0.1"
		if ctx.GlobalIsSet(WSListenAddrFlag.Name) {
			cfg.WSHost = ctx.GlobalString(WSListenAddrFlag.Name)
		}
	}

	if ctx.GlobalIsSet(WSPortFlag.Name) {
		cfg.WSPort = ctx.GlobalInt(WSPortFlag.Name)
	}
	if ctx.GlobalIsSet(WSAllowedOriginsFlag.Name) {
		cfg.WSOrigins = splitAndTrim(ctx.GlobalString(WSAllowedOriginsFlag.Name))
	}
	if ctx.GlobalIsSet(WSApiFlag.Name) {
		cfg.WSModules = splitAndTrim(ctx.GlobalString(WSApiFlag.Name))
	}
}

// setIPC creates an IPC path configuration from the set command line flags,
// returning an empty string if IPC was explicitly disabled, or the set path.
func setIPC(ctx *cli.Context, cfg *node.Config) {
	CheckExclusive(ctx, IPCDisabledFlag, IPCPathFlag)
	switch {
	case ctx.GlobalBool(IPCDisabledFlag.Name):
		cfg.IPCPath = ""
	case ctx.GlobalIsSet(IPCPathFlag.Name):
		cfg.IPCPath = ctx.GlobalString(IPCPathFlag.Name)
	}
}

// makeDatabaseHandles raises out the number of allowed file handles per process
// for Geth and returns half of the allowance to assign to the database.
func makeDatabaseHandles() int {
	limit, err := fdlimit.Maximum()
	if err != nil {
		Fatalf("Failed to retrieve file descriptor allowance: %v", err)
	}
	raised, err := fdlimit.Raise(uint64(limit))
	if err != nil {
		Fatalf("Failed to raise file descriptor allowance: %v", err)
	}
	return int(raised / 2) // Leave half for networking and other stuff
}

// MakeAddress converts an account specified directly as a hex encoded string or
// a key index in the key store to an internal account representation.
func MakeAddress(ks *keystore.KeyStore, account string) (accounts.Account, error) {
	// If the specified account is a valid address, return it
	if common.IsBech32Address(account) {
		return accounts.Account{Address: common.MustBech32ToAddress(account)}, nil
	}
	if common.IsHexAddress(account) {
		return accounts.Account{Address: common.HexToAddress(account)}, nil
	}
	// Otherwise try to interpret the account as a keystore index
	index, err := strconv.Atoi(account)
	if err != nil || index < 0 {
		return accounts.Account{}, fmt.Errorf("invalid account address or index %q", account)
	}
	log.Warn("-------------------------------------------------------------------")
	log.Warn("Referring to accounts by order in the keystore folder is dangerous!")
	log.Warn("This functionality is deprecated and will be removed in the future!")
	log.Warn("Please use explicit addresses! (can search via `alaya account list`)")
	log.Warn("-------------------------------------------------------------------")

	accs := ks.Accounts()
	if len(accs) <= index {
		return accounts.Account{}, fmt.Errorf("index %d higher than number of accounts %d", index, len(accs))
	}
	return accs[index], nil
}

// MakePasswordList reads password lines from the file specified by the global --password flag.
func MakePasswordList(ctx *cli.Context) []string {
	path := ctx.GlobalString(PasswordFileFlag.Name)
	if path == "" {
		return nil
	}
	text, err := ioutil.ReadFile(path)
	if err != nil {
		Fatalf("Failed to read password file: %v", err)
	}
	lines := strings.Split(string(text), "\n")
	// Sanitise DOS line endings.
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	return lines
}

func SetP2PConfig(ctx *cli.Context, cfg *p2p.Config) {
	setNodeKey(ctx, cfg)
	setNAT(ctx, cfg)
	setListenAddress(ctx, cfg)
	setBootstrapNodes(ctx, cfg)
	// setBootstrapNodesV5(ctx, cfg)

	lightClient := ctx.GlobalString(SyncModeFlag.Name) == "light"
	lightServer := ctx.GlobalInt(LightServFlag.Name) != 0
	lightPeers := ctx.GlobalInt(LightPeersFlag.Name)

	if ctx.GlobalIsSet(MaxConsensusPeersFlag.Name) {
		cfg.MaxConsensusPeers = ctx.GlobalInt(MaxConsensusPeersFlag.Name)
	}

	if ctx.GlobalIsSet(MaxPeersFlag.Name) {
		cfg.MaxPeers = ctx.GlobalInt(MaxPeersFlag.Name)
		if lightServer && !ctx.GlobalIsSet(LightPeersFlag.Name) {
			cfg.MaxPeers += lightPeers
		}
	} else {
		if lightServer {
			cfg.MaxPeers += lightPeers
		}
		if lightClient && ctx.GlobalIsSet(LightPeersFlag.Name) && cfg.MaxPeers < lightPeers {
			cfg.MaxPeers = lightPeers
		}
	}
	if !(lightClient || lightServer) {
		lightPeers = 0
	}
	ethPeers := cfg.MaxPeers - lightPeers
	if lightClient {
		ethPeers = 0
	}
	if cfg.MaxPeers <= cfg.MaxConsensusPeers {
		log.Error("MaxPeers is less than MaxConsensusPeers", "MaxPeers", cfg.MaxPeers, "MaxConsensusPeers", cfg.MaxConsensusPeers)
		Fatalf("MaxPeers is less than MaxConsensusPeers, MaxPeers: %d, MaxConsensusPeers: %d", cfg.MaxPeers, cfg.MaxConsensusPeers)
	}
	log.Info("Maximum peer count", "ETH", ethPeers, "LES", lightPeers, "consensusTotal", cfg.MaxConsensusPeers, "total", cfg.MaxPeers)

	if ctx.GlobalIsSet(MaxPendingPeersFlag.Name) {
		cfg.MaxPendingPeers = ctx.GlobalInt(MaxPendingPeersFlag.Name)
	}
	if ctx.GlobalIsSet(NoDiscoverFlag.Name) || lightClient {
		cfg.NoDiscovery = true
	}

	// if we're running a light client or server, force enable the v5 peer discovery
	// unless it is explicitly disabled with --nodiscover note that explicitly specifying
	// --v5disc overrides --nodiscover, in which case the later only disables v4 discovery
	forceV5Discovery := (lightClient || lightServer) && !ctx.GlobalBool(NoDiscoverFlag.Name)
	if forceV5Discovery {
		cfg.DiscoveryV5 = true
	}
	/*
		if ctx.GlobalIsSet(DiscoveryV5Flag.Name) {
			cfg.DiscoveryV5 = ctx.GlobalBool(DiscoveryV5Flag.Name)
		} else if forceV5Discovery {
			cfg.DiscoveryV5 = true
		}*/

	if netrestrict := ctx.GlobalString(NetrestrictFlag.Name); netrestrict != "" {
		list, err := netutil.ParseNetlist(netrestrict)
		if err != nil {
			Fatalf("Option %q: %v", NetrestrictFlag.Name, err)
		}
		cfg.NetRestrict = list
	}

}

// SetNodeConfig applies node-related command line flags to the config.
func SetNodeConfig(ctx *cli.Context, cfg *node.Config) {
	SetP2PConfig(ctx, &cfg.P2P)
	setIPC(ctx, cfg)
	setHTTP(ctx, cfg)
	setWS(ctx, cfg)
	setNodeUserIdent(ctx, cfg)

	switch {
	case ctx.GlobalIsSet(DataDirFlag.Name):
		cfg.DataDir = ctx.GlobalString(DataDirFlag.Name)
	case ctx.GlobalBool(AlayaNetFlag.Name):
		cfg.DataDir = filepath.Join(node.DefaultDataDir(), "alayanet")
	}

	if ctx.GlobalIsSet(KeyStoreDirFlag.Name) {
		cfg.KeyStoreDir = ctx.GlobalString(KeyStoreDirFlag.Name)
	}
	if ctx.GlobalIsSet(LightKDFFlag.Name) {
		cfg.UseLightweightKDF = ctx.GlobalBool(LightKDFFlag.Name)
	}
	if ctx.GlobalIsSet(NoUSBFlag.Name) {
		cfg.NoUSB = ctx.GlobalBool(NoUSBFlag.Name)
	}
	if ctx.GlobalIsSet(InsecureUnlockAllowedFlag.Name) {
		cfg.InsecureUnlockAllowed = ctx.GlobalBool(InsecureUnlockAllowedFlag.Name)
	}
}

func setGPO(ctx *cli.Context, cfg *gasprice.Config) {
	if ctx.GlobalIsSet(GpoBlocksFlag.Name) {
		cfg.Blocks = ctx.GlobalInt(GpoBlocksFlag.Name)
	}
	if ctx.GlobalIsSet(GpoPercentileFlag.Name) {
		cfg.Percentile = ctx.GlobalInt(GpoPercentileFlag.Name)
	}
}

func setTxPool(ctx *cli.Context, cfg *core.TxPoolConfig) {
	if ctx.GlobalIsSet(TxPoolLocalsFlag.Name) {
		locals := strings.Split(ctx.GlobalString(TxPoolLocalsFlag.Name), ",")
		for _, account := range locals {
			if trimmed := strings.TrimSpace(account); !common.IsBech32Address(trimmed) {
				Fatalf("Invalid account in --txpool.locals: %s", trimmed)
			} else {
				cfg.Locals = append(cfg.Locals, common.MustBech32ToAddress(account))
			}
		}
	}
	if ctx.GlobalIsSet(TxPoolNoLocalsFlag.Name) {
		cfg.NoLocals = ctx.GlobalBool(TxPoolNoLocalsFlag.Name)
	}
	if ctx.GlobalIsSet(TxPoolJournalFlag.Name) {
		cfg.Journal = ctx.GlobalString(TxPoolJournalFlag.Name)
	}
	if ctx.GlobalIsSet(TxPoolRejournalFlag.Name) {
		cfg.Rejournal = ctx.GlobalDuration(TxPoolRejournalFlag.Name)
	}
	if ctx.GlobalIsSet(TxPoolPriceBumpFlag.Name) {
		cfg.PriceBump = ctx.GlobalUint64(TxPoolPriceBumpFlag.Name)
	}
	if ctx.GlobalIsSet(TxPoolAccountSlotsFlag.Name) {
		cfg.AccountSlots = ctx.GlobalUint64(TxPoolAccountSlotsFlag.Name)
	}
	if ctx.GlobalIsSet(TxPoolGlobalSlotsFlag.Name) {
		cfg.GlobalSlots = ctx.GlobalUint64(TxPoolGlobalSlotsFlag.Name)
	}
	if ctx.GlobalIsSet(TxPoolAccountQueueFlag.Name) {
		cfg.AccountQueue = ctx.GlobalUint64(TxPoolAccountQueueFlag.Name)
	}
	if ctx.GlobalIsSet(TxPoolGlobalQueueFlag.Name) {
		cfg.GlobalQueue = ctx.GlobalUint64(TxPoolGlobalQueueFlag.Name)
	}
	if ctx.GlobalIsSet(TxPoolGlobalTxCountFlag.Name) {
		cfg.GlobalTxCount = ctx.GlobalUint64(TxPoolGlobalTxCountFlag.Name)
	}
	if ctx.GlobalIsSet(TxPoolLifetimeFlag.Name) {
		cfg.Lifetime = ctx.GlobalDuration(TxPoolLifetimeFlag.Name)
	}
	if ctx.GlobalIsSet(TxPoolCacheSizeFlag.Name) {
		cfg.TxCacheSize = ctx.GlobalUint64(TxPoolCacheSizeFlag.Name)
	}
}

//func setMpcPool(ctx *cli.Context, cfg *core.MPCPoolConfig) {
//	if ctx.GlobalIsSet(MPCEnabledFlag.Name) {
//		cfg.MPCEnable = ctx.GlobalBool(MPCEnabledFlag.Name)
//	}
//	if ctx.GlobalIsSet(MPCActorFlag.Name) {
//		cfg.MpcActor = common.HexToAddress(ctx.GlobalString(MPCActorFlag.Name))
//	}
//	if file := ctx.GlobalString(MPCIceFileFlag.Name); file != "" {
//		if _, err := os.Stat(file); err != nil {
//			fmt.Println("ice conf not exists.")
//			return
//		}
//		if b := filepath.IsAbs(file); !b {
//			absPath, err := filepath.Abs(file)
//			if err != nil {
//				fmt.Println("Read abs path of ice conf fail: ", err.Error())
//				return
//			}
//			cfg.IceConf = absPath
//		} else {
//			cfg.IceConf = file
//		}
//	}
//}
//
//func setVcPool(ctx *cli.Context, cfg *core.VCPoolConfig) {
//	if ctx.GlobalIsSet(VCEnabledFlag.Name) {
//		cfg.VCEnable = ctx.GlobalBool(VCEnabledFlag.Name)
//	}
//	if ctx.GlobalIsSet(VCActorFlag.Name) {
//		cfg.VcActor = common.HexToAddress(ctx.GlobalString(VCActorFlag.Name))
//		fmt.Println("cfg.VcActor", cfg.VcActor)
//	}
//
//	if ctx.GlobalIsSet(VCPasswordFlag.Name) {
//		cfg.VcPassword = ctx.GlobalString(VCPasswordFlag.Name)
//	}
//
//}
func setMiner(ctx *cli.Context, cfg *miner.Config) {
	if ctx.GlobalIsSet(MinerGasTargetFlag.Name) {
		cfg.GasFloor = ctx.GlobalUint64(MinerGasTargetFlag.Name)
	}
	if ctx.GlobalIsSet(MinerGasPriceFlag.Name) {
		cfg.GasPrice = GlobalBig(ctx, MinerGasPriceFlag.Name)
	}
}

// SetEthConfig applies eth-related command line flags to the config.
func SetEthConfig(ctx *cli.Context, stack *node.Node, cfg *eth.Config) {
	// Avoid conflicting network flags
	CheckExclusive(ctx, AlayaNetFlag)
	CheckExclusive(ctx, LightServFlag, SyncModeFlag, "light")

	setGPO(ctx, &cfg.GPO)
	setTxPool(ctx, &cfg.TxPool)
	// for mpc compute
	//setMpcPool(ctx, &cfg.MPCPool)
	//setVcPool(ctx, &cfg.VCPool)

	if ctx.GlobalIsSet(CbftWalDisabledFlag.Name) {
		cfg.CbftConfig.WalMode = false
	}
	if ctx.GlobalIsSet(SyncModeFlag.Name) {
		cfg.SyncMode = *GlobalTextMarshaler(ctx, SyncModeFlag.Name).(*downloader.SyncMode)
	}
	if ctx.GlobalIsSet(LightServFlag.Name) {
		cfg.LightServ = ctx.GlobalInt(LightServFlag.Name)
	}
	if ctx.GlobalIsSet(LightPeersFlag.Name) {
		cfg.LightPeers = ctx.GlobalInt(LightPeersFlag.Name)
	}
	if ctx.GlobalIsSet(NetworkIdFlag.Name) {
		cfg.NetworkId = ctx.GlobalUint64(NetworkIdFlag.Name)
	}

	if ctx.GlobalIsSet(CacheFlag.Name) || ctx.GlobalIsSet(CacheDatabaseFlag.Name) {
		cfg.DatabaseCache = ctx.GlobalInt(CacheFlag.Name) * ctx.GlobalInt(CacheDatabaseFlag.Name) / 100
	}
	cfg.DatabaseHandles = makeDatabaseHandles()
	if ctx.GlobalIsSet(AncientFlag.Name) {
		cfg.DatabaseFreezer = ctx.GlobalString(AncientFlag.Name)
	}

	cfg.NoPruning = true

	if ctx.GlobalIsSet(CacheFlag.Name) || ctx.GlobalIsSet(CacheGCFlag.Name) {
		cfg.TrieCache = ctx.GlobalInt(CacheFlag.Name) * ctx.GlobalInt(CacheGCFlag.Name) / 100
	}
	if ctx.GlobalIsSet(CacheTrieDBFlag.Name) {
		cfg.TrieDBCache = ctx.GlobalInt(CacheTrieDBFlag.Name)
	}
	if ctx.GlobalIsSet(DocRootFlag.Name) {
		cfg.DocRoot = ctx.GlobalString(DocRootFlag.Name)
	}

	if ctx.GlobalIsSet(RPCGlobalGasCap.Name) {
		cfg.RPCGasCap = new(big.Int).SetUint64(ctx.GlobalUint64(RPCGlobalGasCap.Name))
	}

	// Override any default configs for hard coded networks.
	switch {
	case ctx.GlobalBool(AlayaNetFlag.Name):
		if !ctx.GlobalIsSet(NetworkIdFlag.Name) {
			cfg.NetworkId = 1
		}
		cfg.Genesis = core.DefaultAlayaGenesisBlock()
	}
	if ctx.GlobalIsSet(DBNoGCFlag.Name) {
		cfg.DBDisabledGC = ctx.GlobalBool(DBNoGCFlag.Name)
	}
	if ctx.GlobalIsSet(DBGCIntervalFlag.Name) {
		cfg.DBGCInterval = ctx.GlobalUint64(DBGCIntervalFlag.Name)
	}
	if ctx.GlobalIsSet(DBGCTimeoutFlag.Name) {
		cfg.DBGCTimeout = ctx.GlobalDuration(DBGCTimeoutFlag.Name)
	}
	if ctx.GlobalIsSet(DBGCMptFlag.Name) {
		cfg.DBGCMpt = ctx.GlobalBool(DBGCMptFlag.Name)
	}
	if ctx.GlobalIsSet(DBGCBlockFlag.Name) {
		b := ctx.GlobalInt(DBGCBlockFlag.Name)
		if b > 0 {
			cfg.DBGCBlock = b
		}
	}

	// vm options
	if ctx.GlobalIsSet(VMWasmType.Name) {
		cfg.VMWasmType = ctx.GlobalString(VMWasmType.Name)
	}
	if ctx.GlobalIsSet(VmTimeoutDuration.Name) {
		cfg.VmTimeoutDuration = ctx.GlobalUint64(VmTimeoutDuration.Name)
	}

}

func SetCbft(ctx *cli.Context, cfg *types.OptionsConfig, nodeCfg *node.Config) {
	if nodeCfg.P2P.PrivateKey != nil {
		cfg.NodePriKey = nodeCfg.P2P.PrivateKey
		cfg.NodeID = discover.PubkeyID(&cfg.NodePriKey.PublicKey)
	}

	if ctx.GlobalIsSet(CbftBlsPriKeyFileFlag.Name) {
		priKey, err := bls.LoadBLS(ctx.GlobalString(CbftBlsPriKeyFileFlag.Name))
		if err != nil {
			Fatalf("Failed to load bls key from file: %v", err)
		}
		cfg.BlsPriKey = priKey
	} else {
		cfg.BlsPriKey = nodeCfg.BlsKey()
	}
	nodeCfg.P2P.BlsPublicKey = *(cfg.BlsPriKey.GetPublicKey())

	if ctx.GlobalIsSet(CbftWalDisabledFlag.Name) {
		cfg.WalMode = !ctx.GlobalBool(CbftWalDisabledFlag.Name)
	}

	if ctx.GlobalIsSet(CbftPeerMsgQueueSize.Name) {
		cfg.PeerMsgQueueSize = ctx.GlobalUint64(CbftPeerMsgQueueSize.Name)
	}

	if ctx.GlobalIsSet(CbftMaxPingLatency.Name) {
		cfg.MaxPingLatency = ctx.GlobalInt64(CbftMaxPingLatency.Name)
	}
	if ctx.GlobalIsSet(CbftBlacklistDeadlineFlag.Name) {
		cfg.BlacklistDeadline = ctx.GlobalInt64(CbftBlacklistDeadlineFlag.Name)
	}

}

// RegisterEthService adds an Ethereum client to the stack.
func RegisterEthService(stack *node.Node, cfg *eth.Config) {
	var err error
	if cfg.SyncMode == downloader.LightSync {
		err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			light, err := les.New(ctx, cfg)
			if err == nil {
				stack.ChainID = light.ApiBackend.ChainConfig().ChainID
			}
			return light, err
		})
	} else {
		err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			fullNode, err := eth.New(ctx, cfg)
			if err == nil {
				stack.ChainID = fullNode.APIBackend.ChainConfig().ChainID
			}

			if fullNode != nil && cfg.LightServ > 0 {
				ls, _ := les.NewLesServer(fullNode, cfg)
				fullNode.AddLesServer(ls)
			}
			return fullNode, err
		})
	}
	if err != nil {
		Fatalf("Failed to register the Alaya-Go service: %v", err)
	}
}

// RegisterShhService configures Whisper and adds it to the given node.
//func RegisterShhService(stack *node.Node, cfg *whisper.Config) {
//	if err := stack.Register(func(n *node.ServiceContext) (node.Service, error) {
//		return whisper.New(cfg), nil
//	}); err != nil {
//		Fatalf("Failed to register the Whisper service: %v", err)
//	}
//}

// RegisterEthStatsService configures the Ethereum Stats daemon and adds it to
// the given node.
func RegisterEthStatsService(stack *node.Node, url string) {
	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		// Retrieve both eth and les services
		var ethServ *eth.Ethereum
		ctx.Service(&ethServ)

		var lesServ *les.LightEthereum
		ctx.Service(&lesServ)

		return ethstats.New(url, ethServ, lesServ)
	}); err != nil {
		Fatalf("Failed to register the Alaya-Go Stats service: %v", err)
	}
}

func SetupMetrics(ctx *cli.Context) {
	if metrics.Enabled {
		log.Info("Enabling metrics collection")
		var (
			enableExport = ctx.GlobalBool(MetricsEnableInfluxDBFlag.Name)
			endpoint     = ctx.GlobalString(MetricsInfluxDBEndpointFlag.Name)
			database     = ctx.GlobalString(MetricsInfluxDBDatabaseFlag.Name)
			username     = ctx.GlobalString(MetricsInfluxDBUsernameFlag.Name)
			password     = ctx.GlobalString(MetricsInfluxDBPasswordFlag.Name)
		)

		if enableExport {
			tagsMap := SplitTagsFlag(ctx.GlobalString(MetricsInfluxDBTagsFlag.Name))

			log.Info("Enabling metrics export to InfluxDB")
			go influxdb.InfluxDBWithTags(metrics.DefaultRegistry, 10*time.Second, endpoint, database, username, password, "alaya.", tagsMap)
		}
	}
}

func SplitTagsFlag(tagsFlag string) map[string]string {
	tags := strings.Split(tagsFlag, ",")
	tagsMap := map[string]string{}

	for _, t := range tags {
		if t != "" {
			kv := strings.Split(t, "=")

			if len(kv) == 2 {
				tagsMap[kv[0]] = kv[1]
			}
		}
	}

	return tagsMap
}

// MakeChainDatabase open an LevelDB using the flags passed to the client and will hard crash if it fails.
func MakeChainDatabase(ctx *cli.Context, stack *node.Node) ethdb.Database {
	var (
		cache   = ctx.GlobalInt(CacheFlag.Name) * ctx.GlobalInt(CacheDatabaseFlag.Name) / 100
		handles = makeDatabaseHandles()
	)
	name := "chaindata"
	if ctx.GlobalString(SyncModeFlag.Name) == "light" {
		name = "lightchaindata"
	}
	chainDb, err := stack.OpenDatabaseWithFreezer(name, cache, handles, ctx.GlobalString(AncientFlag.Name), "")
	if err != nil {
		Fatalf("Could not open database: %v", err)
	}
	return chainDb
}

func MakeGenesis(ctx *cli.Context) *core.Genesis {
	var genesis *core.Genesis
	switch {
	case ctx.GlobalBool(AlayaNetFlag.Name):
		genesis = core.DefaultAlayaGenesisBlock()
	}
	return genesis
}

// MakeChain creates a chain manager from set command line flags.
func MakeChain(ctx *cli.Context, stack *node.Node) (chain *core.BlockChain, chainDb ethdb.Database) {
	var err error
	chainDb = MakeChainDatabase(ctx, stack)
	basedb, err := snapshotdb.Open(stack.ResolvePath(snapshotdb.DBPath), 0, 0, true)
	if err != nil {
		Fatalf("%v", err)
	}
	config, _, err := core.SetupGenesisBlock(chainDb, basedb, MakeGenesis(ctx))
	if err != nil {
		Fatalf("%v", err)
	}
	if err := basedb.Close(); err != nil {
		Fatalf("%v", err)
	}
	var engine consensus.Engine
	//todo: Merge confirmation.
	engine = consensus.NewFaker()

	cache := &core.CacheConfig{
		Disabled:        true,
		TrieDirtyLimit:  eth.DefaultConfig.TrieCache,
		TrieTimeLimit:   eth.DefaultConfig.TrieTimeout,
		BodyCacheLimit:  eth.DefaultConfig.BodyCacheLimit,
		BlockCacheLimit: eth.DefaultConfig.BlockCacheLimit,
		MaxFutureBlocks: eth.DefaultConfig.MaxFutureBlocks,
		BadBlockLimit:   eth.DefaultConfig.BadBlockLimit,
		TriesInMemory:   eth.DefaultConfig.TriesInMemory,
	}
	if ctx.GlobalIsSet(CacheFlag.Name) || ctx.GlobalIsSet(CacheGCFlag.Name) {
		cache.TrieDirtyLimit = ctx.GlobalInt(CacheFlag.Name) * ctx.GlobalInt(CacheGCFlag.Name) / 100
	}
	vmcfg := vm.Config{}
	chain, err = core.NewBlockChain(chainDb, cache, config, engine, vmcfg, nil)
	if err != nil {
		Fatalf("Can't create BlockChain: %v", err)
	}

	return chain, chainDb
}

// MakeChain creates a chain manager from set command line flags.
func MakeChainForCBFT(ctx *cli.Context, stack *node.Node, cfg *eth.Config, nodeCfg *node.Config) (chain *core.BlockChain, chainDb ethdb.Database) {
	var err error
	chainDb = MakeChainDatabase(ctx, stack)
	basedb, err := snapshotdb.Open(stack.ResolvePath(snapshotdb.DBPath), 0, 0, true)
	if err != nil {
		Fatalf("%v", err)
	}
	config, _, err := core.SetupGenesisBlock(chainDb, basedb, MakeGenesis(ctx))
	if err != nil {
		Fatalf("%v", err)
	}
	if err := basedb.Close(); err != nil {
		Fatalf("%v", err)
	}
	var engine consensus.Engine
	if config.Cbft != nil {
		sc := node.NewServiceContext(nodeCfg, nil, stack.EventMux(), stack.AccountManager())
		engine = eth.CreateConsensusEngine(sc, config, false, chainDb, &cfg.CbftConfig, stack.EventMux())
	}

	cache := &core.CacheConfig{
		Disabled:        true,
		TrieDirtyLimit:  eth.DefaultConfig.TrieCache,
		TrieTimeLimit:   eth.DefaultConfig.TrieTimeout,
		BodyCacheLimit:  eth.DefaultConfig.BodyCacheLimit,
		BlockCacheLimit: eth.DefaultConfig.BlockCacheLimit,
		MaxFutureBlocks: eth.DefaultConfig.MaxFutureBlocks,
		BadBlockLimit:   eth.DefaultConfig.BadBlockLimit,
		TriesInMemory:   eth.DefaultConfig.TriesInMemory,
	}
	if ctx.GlobalIsSet(CacheFlag.Name) || ctx.GlobalIsSet(CacheGCFlag.Name) {
		cache.TrieDirtyLimit = ctx.GlobalInt(CacheFlag.Name) * ctx.GlobalInt(CacheGCFlag.Name) / 100
	}
	vmcfg := vm.Config{}
	chain, err = core.NewBlockChain(chainDb, cache, config, engine, vmcfg, nil)
	if err != nil {
		Fatalf("Can't create BlockChain: %v", err)
	}
	return chain, chainDb
}

// MakeConsolePreloads retrieves the absolute paths for the console JavaScript
// scripts to preload before starting.
func MakeConsolePreloads(ctx *cli.Context) []string {
	// Skip preloading if there's nothing to preload
	if ctx.GlobalString(PreloadJSFlag.Name) == "" {
		return nil
	}
	// Otherwise resolve absolute paths and return them
	preloads := []string{}

	assets := ctx.GlobalString(JSpathFlag.Name)
	for _, file := range strings.Split(ctx.GlobalString(PreloadJSFlag.Name), ",") {
		preloads = append(preloads, common.AbsolutePath(assets, strings.TrimSpace(file)))
	}
	return preloads
}

// MigrateFlags sets the global flag from a local flag when it's set.
// This is a temporary function used for migrating old command/flags to the
// new format.
//
// e.g. alaya account new --keystore /tmp/mykeystore --lightkdf
//
// is equivalent after calling this method with:
//
// alaya --keystore /tmp/mykeystore --lightkdf account new
//
// This allows the use of the existing configuration functionality.
// When all flags are migrated this function can be removed and the existing
// configuration functionality must be changed that is uses local flags
func MigrateFlags(action func(ctx *cli.Context) error) func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		for _, name := range ctx.FlagNames() {
			if ctx.IsSet(name) {
				ctx.GlobalSet(name, ctx.String(name))
			}
		}
		return action(ctx)
	}
}

// CheckExclusive verifies that only a single instance of the provided flags was
// set by the user. Each flag might optionally be followed by a string type to
// specialize it further.
func CheckExclusive(ctx *cli.Context, args ...interface{}) {
	set := make([]string, 0, 1)
	for i := 0; i < len(args); i++ {
		// Make sure the next argument is a flag and skip if not set
		flag, ok := args[i].(cli.Flag)
		if !ok {
			panic(fmt.Sprintf("invalid argument, not cli.Flag type: %T", args[i]))
		}
		// Check if next arg extends current and expand its name if so
		name := flag.GetName()

		if i+1 < len(args) {
			switch option := args[i+1].(type) {
			case string:
				// Extended flag check, make sure value set doesn't conflict with passed in option
				if ctx.GlobalString(flag.GetName()) == option {
					name += "=" + option
					set = append(set, "--"+name)
				}
				// shift arguments and continue
				i++
				continue

			case cli.Flag:
			default:
				panic(fmt.Sprintf("invalid argument, not cli.Flag or string extension: %T", args[i+1]))
			}
		}
		// Mark the flag if it's set
		if ctx.GlobalIsSet(flag.GetName()) {
			set = append(set, "--"+name)
		}
	}
	if len(set) > 1 {
		Fatalf("Flags %v can't be used at the same time", strings.Join(set, ", "))
	}
}
