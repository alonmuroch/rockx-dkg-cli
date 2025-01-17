# DKG Operator Node - installation

### Environment variables for .env file
```
NODE_OPERATOR_ID=1
NODE_ADDR=0.0.0.0:8080
NODE_BROADCAST_ADDR=<public ip or public address>
MESSENGER_SRV_ADDR=https://dkg-messenger.rockx.com
KEYSTORE_FILE_PATH=/keys/<keystore file name>
KEYSTORE_PASSWORD=password
USE_HARDCODED_OPERATORS=false
```

> Note: keep USE_HARDCODED_OPERATORS=false to use SSV operator registry instead of hardcoded values


### Docker command to run the containers

#### Upload Keystore file

If you do not have a keystore prepared, you may refer to the section on creating or importing keystore files by clicking [here](#creatingimporting-keystore-files).

To proceed, create a new folder called `keystorefiles` in this location, and upload your keystore file there. 
It's important to ensure that you're using the keystore file corresponding to the eth address of the SSV node operator's owner. 

```
ls keystorefiles/
<keystore_file_name> #output
```
#### Environment Variables file

Create a folder `env` and store your env file here

```
ls env/
operator.1.env
```

#### Pull your docker image from GCP container registry
```
docker pull asia-southeast1-docker.pkg.dev/rockx-mpc-lab/rockx-dkg/rockx-dkg-node:latest
```

#### Run the container with the env file and the keystore folder
```
docker run -d --name operator-node -v $PWD/keystorefiles:/keys --env-file ./env/operator.1.env -p 8080:8080 asia-southeast1-docker.pkg.dev/rockx-mpc-lab/rockx-dkg/rockx-dkg-node
```

### Creating/Importing keystore files

Keystore files (version 3) for ethereum accounts can be generated in multiple ways, here is an example by using a tool called `clef`. 

The documentation and installation instructions for clef can be found here
> https://geth.ethereum.org/docs/tools/clef/introduction

or you can simply run it via docker 

```
mkdir keystorefiles
docker run -it --rm --entrypoint sh -v $PWD/keystorefiles:/keystorefiles ethereum/client-go:alltools-stable
```

Now you can run commands for creating a new account or importing an existing one from below

#### Creating a new key
```
> clef newaccount --keystore <path>

Please enter a password for the new account to be created:
> <password>

------------
INFO [10-28|16:19:09.156] Your new key was generated       address=0x5e97870f263700f46aa00d967821199b9bc5a120
WARN [10-28|16:19:09.306] Please backup your key file      path=/home/user/go-ethereum/data/keystore/UTC--2022-10-28T15-19-08.000825927Z--5e97870f263700f46aa00d967821199b9bc5a120
WARN [10-28|16:19:09.306] Please remember your password!
Generated account 0x5e97870f263700f46aa00d967821199b9bc5a120
```

You can directly import this keystore file into Metamask by following these [instructions](https://support.metamask.io/hc/en-us/articles/360015489331-How-to-import-an-account#h_01G01W0D3TGE72A7ZBV0FMSZX1)

#### Importing existing key

```
# keyfile.txt contains unencrypted private key in hex format
> clef importraw --keystore <path> keyfile.txt

## Password

Please enter a password for the imported account
>
-----------------------
## Password

Please repeat the password you just entered
>
-----------------------
## Info
Key imported:
  Address 0x5e97870f263700f46aa00d967821199b9bc5a120
  Keystore file: go-ethereum/data/keystore/UUTC--2022-10-28T15-19-08.000825927Z--5e97870f263700f46aa00d967821199b9bc5a120
```
