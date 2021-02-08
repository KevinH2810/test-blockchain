# test-blockchain

#How To Use
 ## Setting up env
 - set up the env variable `NODE_ID` in your pc.
 you can use command
 `export` for mac and linux and `setx` for windows. example
 ```bash
 $ export NODE_ID=10111
 ```
 for now, the PORT that we used are static from 10111 - 10114 and 10111 will be used as the master Node.

 ## Create Wallet
 to create a wallet use the command below
 ```bash
 $ go run main.go createwallet
 ```

## Create Blockchain
to create blockchain, use belo command.
```bash
$ go run main.go createwallet -address <ADDRESS_VALUE>
```
after that it should make a folder inside tmp/block_NODE_ID. and copy it to other nodes ex block_10112, block_10113 and block_10114.

## Get Balance Address
```bash
$ go run main.go getbalance -address <ADDRESS_VALUE>
```

## Start Node
to start as a node, make sure that you have already make the blockchain and a wallet. and then execute the command below.

```bash
$ go run main.go startnode -forger <ADDRESS>
```

## Send Transaction
to send a transaction use below command
```bash
$ go run main.go send -from <ADDRESS> -to <ADDRESS> -amount <VALUE>
```
Note: the transaction is still not done yet. it should be holded in a pool before forged with other latest transaction into one block by the stake winner

## Send StakeTx
```bash
$ go run main.go staketx -from <ADDRESS> -amount <VALUE>
```

## PrintChain - See the Block in our chain
```bash
$ go run main.go printchain
```
this will print all the chain in our blockchain from the newest to the oldest block.