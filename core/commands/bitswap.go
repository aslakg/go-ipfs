package commands

import (
	"bytes"
	"fmt"
	"io"

	cmds "github.com/ipfs/go-ipfs/commands"
	e "github.com/ipfs/go-ipfs/core/commands/e"
	bitswap "github.com/ipfs/go-ipfs/exchange/bitswap"
	decision "github.com/ipfs/go-ipfs/exchange/bitswap/decision"
	cmdsutil "gx/ipfs/Qmf7G7FikwUsm48Jm4Yw4VBGNZuyRaAMzpWDJcW8V71uV2/go-ipfs-cmdkit"

	"gx/ipfs/QmPSBJL4momYnE7DcUyk2DVhD6rH488ZmHBGLbxNdhU44K/go-humanize"
	cid "gx/ipfs/QmYhQaCYEcaPPjxJX7YcPcVKkQfRy6sJ7B3XmGFk82XYdQ/go-cid"
	peer "gx/ipfs/QmdS9KpbDyPrieswibZhkod1oXqRwZJrUPzxCofAMWpFGq/go-libp2p-peer"
)

var BitswapCmd = &cmds.Command{
	Helptext: cmdsutil.HelpText{
		Tagline:          "Interact with the bitswap agent.",
		ShortDescription: ``,
	},
	Subcommands: map[string]*cmds.Command{
		"wantlist": showWantlistCmd,
		"stat":     bitswapStatCmd,
		"unwant":   unwantCmd,
		"ledger":   ledgerCmd,
	},
}

var unwantCmd = &cmds.Command{
	Helptext: cmdsutil.HelpText{
		Tagline: "Remove a given block from your wantlist.",
	},
	Arguments: []cmdsutil.Argument{
		cmdsutil.StringArg("key", true, true, "Key(s) to remove from your wantlist.").EnableStdin(),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		nd, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmdsutil.ErrNormal)
			return
		}

		if !nd.OnlineMode() {
			res.SetError(errNotOnline, cmdsutil.ErrClient)
			return
		}

		bs, ok := nd.Exchange.(*bitswap.Bitswap)
		if !ok {
			res.SetError(e.TypeErr(bs, nd.Exchange), cmdsutil.ErrNormal)
			return
		}

		var ks []*cid.Cid
		for _, arg := range req.Arguments() {
			c, err := cid.Decode(arg)
			if err != nil {
				res.SetError(err, cmdsutil.ErrNormal)
				return
			}

			ks = append(ks, c)
		}

		bs.CancelWants(ks)
	},
}

var showWantlistCmd = &cmds.Command{
	Helptext: cmdsutil.HelpText{
		Tagline: "Show blocks currently on the wantlist.",
		ShortDescription: `
Print out all blocks currently on the bitswap wantlist for the local peer.`,
	},
	Options: []cmdsutil.Option{
		cmdsutil.StringOption("peer", "p", "Specify which peer to show wantlist for. Default: self."),
	},
	Type: KeyList{},
	Run: func(req cmds.Request, res cmds.Response) {
		nd, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmdsutil.ErrNormal)
			return
		}

		if !nd.OnlineMode() {
			res.SetError(errNotOnline, cmdsutil.ErrClient)
			return
		}

		bs, ok := nd.Exchange.(*bitswap.Bitswap)
		if !ok {
			res.SetError(e.TypeErr(bs, nd.Exchange), cmdsutil.ErrNormal)
			return
		}

		pstr, found, err := req.Option("peer").String()
		if err != nil {
			res.SetError(err, cmdsutil.ErrNormal)
			return
		}
		if found {
			pid, err := peer.IDB58Decode(pstr)
			if err != nil {
				res.SetError(err, cmdsutil.ErrNormal)
				return
			}
			res.SetOutput(&KeyList{bs.WantlistForPeer(pid)})
		} else {
			res.SetOutput(&KeyList{bs.GetWantlist()})
		}
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: KeyListTextMarshaler,
	},
}

var bitswapStatCmd = &cmds.Command{
	Helptext: cmdsutil.HelpText{
		Tagline:          "Show some diagnostic information on the bitswap agent.",
		ShortDescription: ``,
	},
	Type: bitswap.Stat{},
	Run: func(req cmds.Request, res cmds.Response) {
		nd, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmdsutil.ErrNormal)
			return
		}

		if !nd.OnlineMode() {
			res.SetError(errNotOnline, cmdsutil.ErrClient)
			return
		}

		bs, ok := nd.Exchange.(*bitswap.Bitswap)
		if !ok {
			res.SetError(e.TypeErr(bs, nd.Exchange), cmdsutil.ErrNormal)
			return
		}

		st, err := bs.Stat()
		if err != nil {
			res.SetError(err, cmdsutil.ErrNormal)
			return
		}

		res.SetOutput(st)
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			v, err := unwrapOutput(res.Output())
			if err != nil {
				return nil, err
			}

			out, ok := v.(*bitswap.Stat)
			if !ok {
				return nil, e.TypeErr(out, v)
			}
			buf := new(bytes.Buffer)
			fmt.Fprintln(buf, "bitswap status")
			fmt.Fprintf(buf, "\tprovides buffer: %d / %d\n", out.ProvideBufLen, bitswap.HasBlockBufferSize)
			fmt.Fprintf(buf, "\tblocks received: %d\n", out.BlocksReceived)
			fmt.Fprintf(buf, "\tblocks sent: %d\n", out.BlocksSent)
			fmt.Fprintf(buf, "\tdata received: %d\n", out.DataReceived)
			fmt.Fprintf(buf, "\tdata sent: %d\n", out.DataSent)
			fmt.Fprintf(buf, "\tdup blocks received: %d\n", out.DupBlksReceived)
			fmt.Fprintf(buf, "\tdup data received: %s\n", humanize.Bytes(out.DupDataReceived))
			fmt.Fprintf(buf, "\twantlist [%d keys]\n", len(out.Wantlist))
			for _, k := range out.Wantlist {
				fmt.Fprintf(buf, "\t\t%s\n", k.String())
			}
			fmt.Fprintf(buf, "\tpartners [%d]\n", len(out.Peers))
			for _, p := range out.Peers {
				fmt.Fprintf(buf, "\t\t%s\n", p)
			}
			return buf, nil
		},
	},
}

var ledgerCmd = &cmds.Command{
	Helptext: cmdsutil.HelpText{
		Tagline: "Show the current ledger for a peer.",
		ShortDescription: `
The Bitswap decision engine tracks the number of bytes exchanged between IPFS
nodes, and stores this information as a collection of ledgers. This command
prints the ledger associated with a given peer.
`,
	},
	Arguments: []cmdsutil.Argument{
		cmdsutil.StringArg("peer", true, false, "The PeerID (B58) of the ledger to inspect."),
	},
	Type: decision.Receipt{},
	Run: func(req cmds.Request, res cmds.Response) {
		nd, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmdsutil.ErrNormal)
			return
		}

		if !nd.OnlineMode() {
			res.SetError(errNotOnline, cmdsutil.ErrClient)
			return
		}

		bs, ok := nd.Exchange.(*bitswap.Bitswap)
		if !ok {
			res.SetError(e.TypeErr(bs, nd.Exchange), cmdsutil.ErrNormal)
			return
		}

		partner, err := peer.IDB58Decode(req.Arguments()[0])
		if err != nil {
			res.SetError(err, cmdsutil.ErrClient)
			return
		}
		res.SetOutput(bs.LedgerForPeer(partner))
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			v, err := unwrapOutput(res.Output())
			if err != nil {
				return nil, err
			}

			out, ok := v.(*decision.Receipt)
			if !ok {
				return nil, e.TypeErr(out, v)
			}

			buf := new(bytes.Buffer)
			fmt.Fprintf(buf, "Ledger for %s\n"+
				"Debt ratio:\t%f\n"+
				"Exchanges:\t%d\n"+
				"Bytes sent:\t%d\n"+
				"Bytes received:\t%d\n\n",
				out.Peer, out.Value, out.Exchanged,
				out.Sent, out.Recv)
			return buf, nil
		},
	},
}
