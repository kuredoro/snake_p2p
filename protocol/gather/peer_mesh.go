package gather

import (
	"fmt"
	"math/bits"
	"sort"
	"strconv"
	"strings"

	"github.com/libp2p/go-libp2p-core/peer"
)

type peerMesh map[peer.ID]map[peer.ID]struct{}

type peerMeshMod func(peerMesh) bool

func addEdge(from, to peer.ID) peerMeshMod {
	return func(mesh peerMesh) bool {
		if _, exists := mesh[from]; !exists {
			mesh[from] = make(map[peer.ID]struct{})
		}

		mesh[from][to] = struct{}{}
		return true
	}
}

func removeEdge(from, to peer.ID) peerMeshMod {
	return func(mesh peerMesh) bool {
		delete(mesh[from], to)
		return false
	}
}

func addDoubleEdge(from, to peer.ID) peerMeshMod {
	return func(mesh peerMesh) bool {
		addEdge(from, to)(mesh)
		addEdge(to, from)(mesh)
		return true
	}
}

func removeDoubleEdge(from, to peer.ID) peerMeshMod {
	return func(mesh peerMesh) bool {
		removeEdge(from, to)(mesh)
		removeEdge(to, from)(mesh)
		return false
	}
}

func removePeer(p peer.ID) peerMeshMod {
	return func(mesh peerMesh) bool {
		neighs, exists := mesh[p]
		if !exists {
			return false
		}

		for n := range neighs {
			delete(mesh[n], p)
		}

		delete(mesh, p)
		return false
	}
}

func (m peerMesh) String() string {
	var str strings.Builder

	index2peer := make([]peer.ID, 0, len(m))
	for id := range m {
		index2peer = append(index2peer, id)
	}

	sort.Slice(index2peer, func(i, j int) bool {
		return index2peer[i].String() < index2peer[j].String()
	})

	peer2index := make(map[peer.ID]int)
	for i, id := range index2peer {
		peer2index[id] = i
	}

	neightbours := make([]int, 0, len(m))
	for i, srcID := range index2peer {
		idStr := srcID.String()
		str.WriteString(strconv.Itoa(i))
		str.WriteRune(' ')
		str.WriteString(idStr[len(idStr)-6:])
		str.WriteString(": ")

		neightbours = neightbours[:0]
		for destID := range m[srcID] {
			neightbours = append(neightbours, peer2index[destID])
		}

		sort.Ints(neightbours)

		for _, index := range neightbours {
			str.WriteString(strconv.Itoa(index))
			str.WriteRune(' ')
		}
		str.WriteRune('\n')
	}

	return str.String()
}

func (m peerMesh) FindClique(n int, required peer.ID) []peer.ID {
	clique := make([]peer.ID, 0, n)

	neighbours := make([]peer.ID, 0, len(m[required]))
	for id := range m[required] {
		neighbours = append(neighbours, id)
	}

	fmt.Printf("neigh %v\n", neighbours)

	// This is unbelievably stupid, but yeah it's O(2^V)
	// TODO: replace with more efficient version.
	for i := uint32(0); i < 1<<len(neighbours); i++ {
		if bits.OnesCount32(i) != n-1 {
			continue
		}

		clique = clique[:0]
		clique = append(clique, required)

		j := i
		for peerIndex := 0; j != 0; j, peerIndex = j>>1, peerIndex+1 {
			if j&1 == 1 {
				clique = append(clique, neighbours[peerIndex])
			}
		}

		nice := make([]string, len(clique))
		for i, id := range clique {
			str := id.Pretty()
			nice[i] = str[len(str)-6:]
		}
		fmt.Printf("clique %v\n", nice)

		if m.IsClique(clique) {
			return clique
		}
	}

	return nil
}

func (m peerMesh) IsClique(nodes []peer.ID) bool {
	ours := make(map[peer.ID]struct{}, len(nodes))
	for _, id := range nodes {
		ours[id] = struct{}{}
	}

	for _, src := range nodes {
		count := 0
		for dst := range m[src] {
			// This is bogus... but possible...
			if src == dst {
				continue
			}

			if _, ok := ours[dst]; !ok {
				continue
			}

			count++
		}

		if count != len(nodes)-1 {
			return false
		}
	}

	return true
}
