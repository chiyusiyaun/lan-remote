package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"syscall"
	"time"
)

const DefaultPort = 8766

type Beacon struct {
	Name    string `json:"name"`
	IP      string `json:"ip"`
	HTTP    int    `json:"http"`
	Version string `json:"version"`
	PINSet  bool   `json:"pin_set"`
}

type Peer struct {
	Name    string    `json:"name"`
	Addr    string    `json:"addr"` // host:port for http
	IP      string    `json:"ip"`
	Version string    `json:"version"`
	Seen    time.Time `json:"seen"`
}

type Service struct {
	port    int
	self    Beacon
	mu      sync.Mutex
	peers   map[string]Peer // key: ip
	stop    chan struct{}
	conn    *net.UDPConn
}

func New(self Beacon, port int) *Service {
	if port <= 0 {
		port = DefaultPort
	}
	if self.Version == "" {
		self.Version = "1.0"
	}
	return &Service{
		port:  port,
		self:  self,
		peers: map[string]Peer{},
		stop:  make(chan struct{}),
	}
}

func (s *Service) SetSelf(b Beacon) {
	s.mu.Lock()
	s.self = b
	s.mu.Unlock()
}

func (s *Service) Peers() []Peer {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Peer, 0, len(s.peers))
	cutoff := time.Now().Add(8 * time.Second)
	for _, p := range s.peers {
		if p.Seen.Before(cutoff) {
			continue // stale
		}
		out = append(out, p)
	}
	return out
}

func (s *Service) Start() error {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var serr error
			if err := c.Control(func(fd uintptr) {
				serr = setReuse(fd)
			}); err != nil {
				return err
			}
			return serr
		},
	}
	pc, err := lc.ListenPacket(context.Background(), "udp4", fmt.Sprintf(":%d", s.port))
	if err != nil {
		// retry without reuse on odd platforms
		pc, err = net.ListenPacket("udp4", fmt.Sprintf(":%d", s.port))
		if err != nil {
			return err
		}
	}
	conn, ok := pc.(*net.UDPConn)
	if !ok {
		return fmt.Errorf("not a UDP conn")
	}
	s.conn = conn
	_ = setBroadcast(conn)

	go s.readLoop()
	go s.announceLoop()
	return nil
}

func (s *Service) Stop() {
	close(s.stop)
	if s.conn != nil {
		_ = s.conn.Close()
	}
}

func (s *Service) announceLoop() {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	dst := &net.UDPAddr{IP: net.IPv4bcast, Port: s.port}
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			s.mu.Lock()
			b := s.self
			s.mu.Unlock()
			b.IP = primaryIP()
			data, _ := json.Marshal(b)
			if s.conn != nil {
				_, _ = s.conn.WriteToUDP(data, dst)
				// also localhost for same-machine testing
				_, _ = s.conn.WriteToUDP(data, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: s.port})
			}
		}
	}
}

func (s *Service) readLoop() {
	buf := make([]byte, 4096)
	for {
		select {
		case <-s.stop:
			return
		default:
		}
		n, from, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-s.stop:
				return
			default:
				time.Sleep(200 * time.Millisecond)
				continue
			}
		}
		var b Beacon
		if err := json.Unmarshal(buf[:n], &b); err != nil {
			continue
		}
		if b.HTTP <= 0 || b.IP == "" {
			continue
		}
		s.mu.Lock()
		if s.self.Name != "" && b.Name == s.self.Name && b.IP == primaryIP() {
			s.mu.Unlock()
			continue // self
		}
		s.peers[b.IP] = Peer{
			Name:    b.Name,
			Addr:    net.JoinHostPort(b.IP, itoa(b.HTTP)),
			IP:      b.IP,
			Version: b.Version,
			Seen:    time.Now(),
		}
		s.mu.Unlock()
		_ = from
	}
}

func primaryIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if ip[0] == 169 && ip[1] == 254 {
				continue
			}
			return ip.String()
		}
	}
	return "127.0.0.1"
}

func itoa(i int) string {
	// tiny helper to avoid importing strconv in hot path docs
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}

func setBroadcast(c *net.UDPConn) error {
	// best-effort; most OSes allow broadcast on UDP by default
	rc, err := c.SyscallConn()
	if err != nil {
		return err
	}
	var serr error
	_ = rc.Control(func(fd uintptr) {
		serr = setSockBroadcast(fd)
	})
	return serr
}

func init() {
	_ = log.Println
}

func PrimaryIP() string { return primaryIP() }
