package bee

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beeclient/debugapi"
	bmtlegacy "github.com/ethersphere/bmt/legacy"
)

// Client manages communication with the Bee node
type Client struct {
	api   *api.Client
	debug *debugapi.Client
}

// ClientOptions holds optional parameters for the Client.
type ClientOptions struct {
	APIURL              *url.URL
	APIInsecureTLS      bool
	DebugAPIURL         *url.URL
	DebugAPIInsecureTLS bool
}

// NewClient returns Bee client
func NewClient(opts ClientOptions) (c *Client) {
	c = &Client{}

	if opts.APIURL != nil {
		c.api = api.NewClient(opts.APIURL, &api.ClientOptions{HTTPClient: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.APIInsecureTLS},
		}}})
	}
	if opts.DebugAPIURL != nil {
		c.debug = debugapi.NewClient(opts.DebugAPIURL, &debugapi.ClientOptions{HTTPClient: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.DebugAPIInsecureTLS},
		}}})
	}

	return
}

// Addresses represents node's addresses
type Addresses struct {
	Overlay   swarm.Address
	Underlay  []string
	Ethereum  string
	PublicKey string
}

// Addresses returns node's addresses
func (c *Client) Addresses(ctx context.Context) (resp Addresses, err error) {
	a, err := c.debug.Node.Addresses(ctx)
	if err != nil {
		return Addresses{}, fmt.Errorf("get addresses: %w", err)
	}

	return Addresses{
		Ethereum:  a.Ethereum,
		Overlay:   a.Overlay,
		PublicKey: a.PublicKey,
		Underlay:  a.Underlay,
	}, nil
}

// Balance represents node's balance with peer
type Balance struct {
	Balance int
	Peer    string
}

// Balance returns node's balance with a given peer
func (c *Client) Balance(ctx context.Context, a swarm.Address) (resp Balance, err error) {
	b, err := c.debug.Node.Balance(ctx, a)
	if err != nil {
		return Balance{}, fmt.Errorf("get balance with node %s: %w", a.String(), err)
	}

	return Balance{
		Balance: b.Balance,
		Peer:    b.Peer,
	}, nil
}

// Balances represents Balances's response
type Balances struct {
	Balances []Balance
}

// Balances returns node's balances
func (c *Client) Balances(ctx context.Context) (resp Balances, err error) {
	r, err := c.debug.Node.Balances(ctx)
	if err != nil {
		return Balances{}, fmt.Errorf("get balances: %w", err)
	}

	for _, b := range r.Balances {
		resp.Balances = append(resp.Balances, Balance{
			Balance: b.Balance,
			Peer:    b.Peer,
		})
	}

	return
}

// DownloadBytes downloads chunk from the node
func (c *Client) DownloadBytes(ctx context.Context, a swarm.Address) (data []byte, err error) {
	r, err := c.api.Bytes.Download(ctx, a)
	if err != nil {
		return nil, fmt.Errorf("download chunk %s: %w", a, err)
	}

	return ioutil.ReadAll(r)
}

// DownloadChunk downloads chunk from the node
func (c *Client) DownloadChunk(ctx context.Context, a swarm.Address, targets string) (data []byte, err error) {
	r, err := c.api.Chunks.Download(ctx, a, targets)
	if err != nil {
		return nil, fmt.Errorf("download chunk %s: %w", a, err)
	}

	return ioutil.ReadAll(r)
}

// DownloadFile downloads chunk from the node and returns it's size and hash
func (c *Client) DownloadFile(ctx context.Context, a swarm.Address) (size int64, hash []byte, err error) {
	r, err := c.api.Files.Download(ctx, a)
	if err != nil {
		return 0, nil, fmt.Errorf("download file %s: %w", a, err)
	}

	h := fileHahser()
	size, err = io.Copy(h, r)
	if err != nil {
		return 0, nil, fmt.Errorf("download file %s, hashing copy: %w", a, err)
	}

	return size, h.Sum(nil), nil
}

// HasChunk returns true/false if node has a chunk
func (c *Client) HasChunk(ctx context.Context, a swarm.Address) (bool, error) {
	return c.debug.Node.HasChunk(ctx, a)
}

// Overlay returns node's overlay address
func (c *Client) Overlay(ctx context.Context) (swarm.Address, error) {
	a, err := c.debug.Node.Addresses(ctx)
	if err != nil {
		return swarm.Address{}, fmt.Errorf("get overlay: %w", err)
	}

	return a.Overlay, nil
}

// Peers returns addresses of node's peers
func (c *Client) Peers(ctx context.Context) (peers []swarm.Address, err error) {
	ps, err := c.debug.Node.Peers(ctx)
	if err != nil {
		return nil, fmt.Errorf("get peers: %w", err)
	}

	for _, p := range ps.Peers {
		peers = append(peers, p.Address)
	}

	return
}

// PinChunk returns true/false if chunk pinning is successful
func (c *Client) PinChunk(ctx context.Context, a swarm.Address) (bool, error) {
	return c.api.Pinning.PinChunk(ctx, a)
}

// PinnedChunk represents pinned chunk
type PinnedChunk struct {
	Address    swarm.Address
	PinCounter int
}

// PinnedChunk returns pinned chunk
func (c *Client) PinnedChunk(ctx context.Context, a swarm.Address) (PinnedChunk, error) {
	p, err := c.api.Pinning.PinnedChunk(ctx, a)
	if err != nil {
		return PinnedChunk{}, fmt.Errorf("get pinned chunk: %w", err)
	}

	return PinnedChunk{
		Address:    p.Address,
		PinCounter: p.PinCounter,
	}, nil
}

// PinnedChunks represents pinned chunks
type PinnedChunks struct {
	Chunks []PinnedChunk
}

// PinnedChunks returns pinned chunks
func (c *Client) PinnedChunks(ctx context.Context) (PinnedChunks, error) {
	p, err := c.api.Pinning.PinnedChunks(ctx)
	if err != nil {
		return PinnedChunks{}, fmt.Errorf("get pinned chunks: %w", err)
	}

	r := PinnedChunks{}
	for _, c := range p.Chunks {
		r.Chunks = append(r.Chunks, PinnedChunk{
			Address:    c.Address,
			PinCounter: c.PinCounter,
		})
	}

	return r, nil
}

// Ping pings other node
func (c *Client) Ping(ctx context.Context, node swarm.Address) (rtt string, err error) {
	r, err := c.debug.PingPong.Ping(ctx, node)
	if err != nil {
		return "", fmt.Errorf("ping node %s: %w", node, err)
	}
	return r.RTT, nil
}

// PingStreamMsg represents message sent over the PingStream channel
type PingStreamMsg struct {
	Node  swarm.Address
	RTT   string
	Index int
	Error error
}

// PingStream returns stream of ping results for given nodes
func (c *Client) PingStream(ctx context.Context, nodes []swarm.Address) <-chan PingStreamMsg {
	pingStream := make(chan PingStreamMsg)

	var wg sync.WaitGroup
	for i, node := range nodes {
		wg.Add(1)
		go func(i int, node swarm.Address) {
			defer wg.Done()

			rtt, err := c.Ping(ctx, node)
			pingStream <- PingStreamMsg{
				Node:  node,
				RTT:   rtt,
				Index: i,
				Error: err,
			}
		}(i, node)
	}

	go func() {
		wg.Wait()
		close(pingStream)
	}()

	return pingStream
}

// Settlement represents node's settlement with peer
type Settlement struct {
	Peer     string
	Received int
	Sent     int
}

// Settlement returns node's settlement with a given peer
func (c *Client) Settlement(ctx context.Context, a swarm.Address) (resp Settlement, err error) {
	b, err := c.debug.Node.Settlement(ctx, a)
	if err != nil {
		return Settlement{}, fmt.Errorf("get settlement with node %s: %w", a.String(), err)
	}

	return Settlement{
		Peer:     b.Peer,
		Received: b.Received,
		Sent:     b.Sent,
	}, nil
}

// Settlements represents Settlements's response
type Settlements struct {
	Settlements   []Settlement
	TotalReceived int
	TotalSent     int
}

// Settlements returns node's settlements
func (c *Client) Settlements(ctx context.Context) (resp Settlements, err error) {
	r, err := c.debug.Node.Settlements(ctx)
	if err != nil {
		return Settlements{}, fmt.Errorf("get settlements: %w", err)
	}

	for _, b := range r.Settlements {
		resp.Settlements = append(resp.Settlements, Settlement{
			Peer:     b.Peer,
			Received: b.Received,
			Sent:     b.Sent,
		})
	}
	resp.TotalReceived = r.TotalReceived
	resp.TotalSent = r.TotalSent

	return
}

// Topology represents Kademlia topology
type Topology struct {
	Overlay        swarm.Address
	Connected      int
	Population     int
	NnLowWatermark int
	Depth          int
	Bins           map[string]Bin
}

// Bin represents Kademlia bin
type Bin struct {
	Connected         int
	ConnectedPeers    []swarm.Address
	DisconnectedPeers []swarm.Address
	Population        int
}

// Topology returns Kademlia topology
func (c *Client) Topology(ctx context.Context) (topology Topology, err error) {
	t, err := c.debug.Node.Topology(ctx)
	if err != nil {
		return Topology{}, fmt.Errorf("get topology: %w", err)
	}

	topology = Topology{
		Overlay:        t.BaseAddr,
		Connected:      t.Connected,
		Population:     t.Population,
		NnLowWatermark: t.NnLowWatermark,
		Depth:          t.Depth,
		Bins:           make(map[string]Bin),
	}

	for k, b := range t.Bins {
		if b.Population > 0 {
			topology.Bins[k] = Bin{
				Connected:         b.Connected,
				ConnectedPeers:    b.ConnectedPeers,
				DisconnectedPeers: b.DisconnectedPeers,
				Population:        b.Population,
			}
		}
	}

	return
}

// Underlay returns node's underlay addresses
func (c *Client) Underlay(ctx context.Context) ([]string, error) {
	a, err := c.debug.Node.Addresses(ctx)
	if err != nil {
		return nil, fmt.Errorf("get underlay: %w", err)
	}

	return a.Underlay, nil
}

// UnpinChunk returns true/false if chunk unpinning is successful
func (c *Client) UnpinChunk(ctx context.Context, a swarm.Address) (bool, error) {
	return c.api.Pinning.UnpinChunk(ctx, a)
}

// UploadBytes uploads bytes to the node
func (c *Client) UploadBytes(ctx context.Context, b []byte, o api.UploadOptions) (swarm.Address, error) {
	r, err := c.api.Bytes.Upload(ctx, bytes.NewReader(b), o)
	if err != nil {
		return swarm.ZeroAddress, fmt.Errorf("upload chunk: %w", err)
	}

	return r.Reference, nil
}

// UploadChunk uploads chunk to the node
func (c *Client) UploadChunk(ctx context.Context, chunk *Chunk, o api.UploadOptions) (err error) {
	p := bmtlegacy.NewTreePool(chunkHahser, swarm.Branches, bmtlegacy.PoolSize)
	hasher := bmtlegacy.New(p)
	err = hasher.SetSpan(int64(chunk.Span()))
	if err != nil {
		return fmt.Errorf("upload chunk: %w", err)
	}
	_, err = hasher.Write(chunk.Data()[8:])
	if err != nil {
		return fmt.Errorf("upload chunk: %w", err)
	}
	chunk.address = swarm.NewAddress(hasher.Sum(nil))

	_, err = c.api.Chunks.Upload(ctx, chunk.address, bytes.NewReader(chunk.Data()), o)
	if err != nil {
		return fmt.Errorf("upload chunk: %w", err)
	}

	return
}

// RemoveChunk removes the given chunk from the node's local store
func (c *Client) RemoveChunk(ctx context.Context, chunk *Chunk) (err error) {
	return c.debug.Chunks.Remove(ctx, chunk.Address())
}

// UploadFile uploads file to the node
func (c *Client) UploadFile(ctx context.Context, f *File, pin bool) (err error) {
	h := fileHahser()
	r, err := c.api.Files.Upload(ctx, f.Name(), io.TeeReader(f.DataReader(), h), f.Size(), pin, 0)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}

	f.address = r.Reference
	f.hash = h.Sum(nil)

	return
}

// UploadFileWithTag uploads file with tag to the node
func (c *Client) UploadFileWithTag(ctx context.Context, f *File, pin bool, tagUID uint32) (err error) {
	h := fileHahser()
	r, err := c.api.Files.Upload(ctx, f.Name(), io.TeeReader(f.DataReader(), h), f.Size(), pin, tagUID)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}

	f.address = r.Reference
	f.hash = h.Sum(nil)

	return
}

// UploadCollection uploads TAR collection bytes to the node
func (c *Client) UploadCollection(ctx context.Context, f *File) (err error) {
	h := fileHahser()
	r, err := c.api.Dirs.Upload(ctx, io.TeeReader(f.DataReader(), h), f.Size())
	if err != nil {
		return fmt.Errorf("upload collection: %w", err)
	}

	f.address = r.Reference
	f.hash = h.Sum(nil)

	return
}

// DownloadManifestFile downloads manifest file from the node and returns it's size and hash
func (c *Client) DownloadManifestFile(ctx context.Context, a swarm.Address, path string) (size int64, hash []byte, err error) {
	r, err := c.api.Dirs.Download(ctx, a, path)
	if err != nil {
		return 0, nil, fmt.Errorf("download manifest file %s: %w", path, err)
	}

	h := fileHahser()
	size, err = io.Copy(h, r)
	if err != nil {
		return 0, nil, fmt.Errorf("download manifest file %s: %w", path, err)
	}

	return size, h.Sum(nil), nil
}

// CreateTag creates tag on the node
func (c *Client) CreateTag(ctx context.Context) (resp api.TagResponse, err error) {

	resp, err = c.api.Tags.CreateTag(ctx)
	if err != nil {
		return resp, fmt.Errorf("create tag: %w", err)
	}

	return
}

// GetTag retrieves tag from node
func (c *Client) GetTag(ctx context.Context, tagUID uint32) (resp api.TagResponse, err error) {

	resp, err = c.api.Tags.GetTag(ctx, tagUID)
	if err != nil {
		return resp, fmt.Errorf("get tag: %w", err)
	}

	return
}
