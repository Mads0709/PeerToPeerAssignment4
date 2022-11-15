# PeerToPeerAssignment4
## About
The program can only be run with three peers.
If a peer is about to access the critical section, and just before it accesses the function another peer sends a request, that peer deadlocks. But if the first peer has entered the critical function the program should work.

## How to run the program
To run the program open a terminal and write the following command: 
`go run . 0`
This command will create a peer with port 5000 + 0

Then open two new terminals and write the same command, but with other numbers such as 1 or 2

To make an access request to the critical section, write any message in one of the three terminals
