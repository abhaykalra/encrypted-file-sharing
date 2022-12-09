package client

// CS 161 Project 2

// You MUST NOT change these default imports. ANY additional imports
// may break the autograder!

import (
	"encoding/json"

	userlib "github.com/cs161-staff/project2-userlib"
	"github.com/google/uuid"

	// hex.EncodeToString(...) is useful for converting []byte to string

	// Useful for string manipulation
	"strings"

	// Useful for formatting strings (e.g. `fmt.Sprintf`).
	"fmt"

	// Useful for creating new error messages to return using errors.New("...")
	"errors"

	// Optional.
	_ "strconv"
)

// This serves two purposes: it shows you a few useful primitives,
// and suppresses warnings for imports not being used. It can be
// safely deleted!
func someUsefulThings() {

	// Creates a random UUID.
	randomUUID := uuid.New()

	// Prints the UUID as a string. %v prints the value in a default format.
	// See https://pkg.go.dev/fmt#hdr-Printing for all Golang format string flags.
	userlib.DebugMsg("Random UUID: %v", randomUUID.String())

	// Creates a UUID deterministically, from a sequence of bytes.
	hash := userlib.Hash([]byte("user-structs/alice"))
	deterministicUUID, err := uuid.FromBytes(hash[:16])
	if err != nil {
		// Normally, we would `return err` here. But, since this function doesn't return anything,
		// we can just panic to terminate execution. ALWAYS, ALWAYS, ALWAYS check for errors! Your
		// code should have hundreds of "if err != nil { return err }" statements by the end of this
		// project. You probably want to avoid using panic statements in your own code.
		panic(errors.New("An error occurred while generating a UUID: " + err.Error()))
	}
	userlib.DebugMsg("Deterministic UUID: %v", deterministicUUID.String())

	// Declares a Course struct type, creates an instance of it, and marshals it into JSON.
	type Course struct {
		name      string
		professor []byte
	}

	course := Course{"CS 161", []byte("Nicholas Weaver")}
	courseBytes, err := json.Marshal(course)
	if err != nil {
		panic(err)
	}

	userlib.DebugMsg("Struct: %v", course)
	userlib.DebugMsg("JSON Data: %v", courseBytes)

	// Generate a random private/public keypair.
	// The "_" indicates that we don't check for the error case here.
	var pk userlib.PKEEncKey
	var sk userlib.PKEDecKey
	pk, sk, _ = userlib.PKEKeyGen()
	userlib.DebugMsg("PKE Key Pair: (%v, %v)", pk, sk)

	// Here's an example of how to use HBKDF to generate a new key from an input key.
	// Tip: generate a new key everywhere you possibly can! It's easier to generate new keys on the fly
	// instead of trying to think about all of the ways a key reuse attack could be performed. It's also easier to
	// store one key and derive multiple keys from that one key, rather than
	originalKey := userlib.RandomBytes(16)
	derivedKey, err := userlib.HashKDF(originalKey, []byte("mac-key"))
	if err != nil {
		panic(err)
	}
	userlib.DebugMsg("Original Key: %v", originalKey)
	userlib.DebugMsg("Derived Key: %v", derivedKey)

	// A couple of tips on converting between string and []byte:
	// To convert from string to []byte, use []byte("some-string-here")
	// To convert from []byte to string for debugging, use fmt.Sprintf("hello world: %s", some_byte_arr).
	// To convert from []byte to string for use in a hashmap, use hex.EncodeToString(some_byte_arr).
	// When frequently converting between []byte and string, just marshal and unmarshal the data.
	//
	// Read more: https://go.dev/blog/strings

	// Here's an example of string interpolation!
	_ = fmt.Sprintf("%s_%d", "file", 1)
}

// This is the type definition for the User struct.
// A Go struct is like a Python or Java class - it can have attributes
// (e.g. like the Username attribute) and methods (e.g. like the StoreFile method below).
type User struct {
	Username   string
	Password   string
	SignKey    userlib.DSSignKey
	privateKey userlib.PKEDecKey
	salt       []byte
	rootKey    []byte
	// You can add other attributes here if you want! But note that in order for attributes to
	// be included when this struct is serialized to/from JSON, they must be capitalized.
	// On the flipside, if you have an attribute that you want to be able to access from
	// this struct's methods, but you DON'T want that value to be included in the serialized value
	// of this struct that's stored in datastore, then you can use a "private" variable (e.g. one that
	// begins with a lowercase letter).
}

// NOTE: The following methods have toy (insecure!) implementations.

type Invite struct {
	PublicEncSymmKey []byte
	FilePath         []byte // hash thew file path
}

type File struct {
	sharedWith  []string // file owner - file name (given by owner)
	FileOwner   string
	FileContent LinkedList
	PlaceHolder userlib.UUID
}

type Node struct {
	next  *Node
	value []byte
}

type LinkedList struct {
	head *Node
	tail *Node
}

// storageKey, err := uuid.FromBytes(userlib.Hash([]byte(filename + userdata.Username))[:16])

func InitUser(username string, password string) (userdataptr *User, err error) {
	var userdata User            // create instance of a user
	userdata.Username = username // set username based on input
	userdata.Password = password // set password based on input and encrypt using ____ scheme??
	// look into how salts can be used here
	var salt []byte = userlib.Hash([]byte(username + password))
	userdata.salt = salt
	var pk userlib.PKEEncKey        //make public key
	var sk userlib.PKEDecKey        //make private key
	pk, sk, _ = userlib.PKEKeyGen() // generate both keys

	var rootKey []byte = userlib.Argon2Key([]byte(password), salt, 32) //generate root key

	// var skEncrypted []byte = SymEnc(rootKey , iv []byte, sk) // encrypt sk with root key
	userdata.privateKey = sk // store private key pkdf encryped in datastore
	userdata.rootKey = rootKey

	storageKey, err := uuid.FromBytes(userlib.Hash([]byte(username))[:16])
	contentBytes, err := json.Marshal(userdata)

	// ENCRYPT THE STRUCT
	var encryptedStruct = userlib.SymEnc(rootKey, salt, contentBytes)

	// CREATE DIGITAL SIGNATURE METHODS
	sk, vk, err := userlib.DSKeyGen()

	userdata.SignKey = sk

	userlib.KeystoreSet("USER"+username, vk) // store users public key for signature verification in keystore

	if err != nil {
		return nil, err
	}
	// CREATE THE DIGITAL SIGNATURE
	signature, err := userlib.DSSign(sk, encryptedStruct)

	if err != nil {
		return nil, err
	}

	// ATTATCH THE SIGNATURE TO THE BEGINNING OF THE ENCRYPTED STRUCT
	store := append(signature[:], encryptedStruct[:]...)
	userlib.DatastoreSet(storageKey, store)

	userlib.KeystoreSet(username, pk) // store users public key in keystore against their username as a key

	return &userdata, nil
}

func GetUser(username string, password string) (userdataptr *User, err error) {
	// cross check username password pair against whatever is stored on insecure server
	var userdata User
	// DERIVE THE KEY USED TO STORE THE BYTES IN DATASTORE
	dataKey, err := uuid.FromBytes(userlib.Hash([]byte(username))[:16])
	if err != nil {
		return nil, err
	}
	// GET THE ENCRYPTED+SIGNED USER STRUCT
	dataJSON, ok := userlib.DatastoreGet(dataKey)
	if !ok {
		return nil, errors.New(strings.ToTitle("user not found"))
	}

	// VERIFY THE INTEGRITY OF THE USERSTRUCT
	vk, ok := userlib.KeystoreGet("USER" + username)
	if !ok {
		return nil, errors.New(strings.ToTitle("Verify Key not Found"))
	}
	verfied_error := userlib.DSVerify(vk, dataJSON[256:], dataJSON[0:256])

	if verfied_error != nil {
		return nil, errors.New(strings.ToTitle("Error Verifying Userstruct"))
	}

	// DERIVE THE DECRYPTION METHOD TO DECRYPT THE USER STRUCT
	var salt []byte = userlib.Hash(([]byte(username + password)))
	var rootKey []byte = userlib.Argon2Key([]byte(password), salt, 32) //generate root key

	// DECRYPT THE USER STRUCT
	dataJSON = userlib.SymDec(rootKey, dataJSON[256:])
	userdataptr = &userdata
	err = json.Unmarshal(dataJSON, userdataptr) //You can derive two keys from one key, then use those two keys to encrypt and MAC.
	return userdataptr, err
}

func (userdata *User) StoreFile(filename string, content []byte) (err error) {
	// first check if the user (userdata.username) already has a file w the same name
	key, err := uuid.FromBytes(userlib.Hash([]byte(filename + "-" + userdata.Username))[:16])
	if err != nil {
		return err
	}
	// GENERATE
	download_data, ok := userlib.DatastoreGet(key)

	// if not create a new file struct
	var filedata File
	var filePlacer File
	if !ok {
		// This is going to be a totally new file
		// Create the first layer of abstraction
		randomUUID := uuid.New()
		// Add uuid to 'fake' file
		filedata.PlaceHolder = randomUUID
		contentBytes, err := json.Marshal(filedata)
		if err != nil {
			return err
		}
		// Encrypt fake file
		encrypted_contents := userlib.SymEnc(userlib.Argon2Key([]byte(userdata.Password+filename), userdata.salt, 32), userdata.salt, contentBytes)
		signature, err := userlib.DSSign(userdata.SignKey, encrypted_contents)
		if err != nil {
			return err
		}
		store := append(signature[:], encrypted_contents[:]...)
		// Store the fake file
		userlib.DatastoreSet(key, store)

		// Start storing the real file
		fileContents := LinkedList{}
		content_node := Node{}
		content_node.value = content
		fileContents.head = &content_node
		fileContents.tail = &content_node
		filePlacer.FileContent = fileContents
		filePlacer.FileOwner = userdata.Username

		contentBytes, err = json.Marshal(filePlacer)
		if err != nil {
			return err
		}
		// Encrypt and store the file with the actual contents
		encrypted_contents = userlib.SymEnc(userlib.Argon2Key([]byte(userdata.Password+filename), userdata.salt, 32), userdata.salt, contentBytes)
		signature, err = userlib.DSSign(userdata.SignKey, encrypted_contents)
		if err != nil {
			return err
		}
		store = append(signature[:], encrypted_contents[:]...)

		userlib.DatastoreSet(randomUUID, store)
	} else {
		// decrypt_download data QUESTION
		vk, ok := userlib.KeystoreGet("USER" + userdata.Username)
		if !ok {
			return errors.New(strings.ToTitle("Verify Key not Found"))
		}
		verfied_error := userlib.DSVerify(vk, download_data[256:], download_data[0:256])
		if verfied_error != nil {
			return errors.New(strings.ToTitle("Error Verifying File"))
		}
		decrypted_data := userlib.SymDec(userlib.Argon2Key([]byte(userdata.Password+filename), userdata.salt, 32), download_data[256:])
		if err != nil {
			return err
		}
		err = json.Unmarshal(decrypted_data, &filedata)

		useUUID := filedata.PlaceHolder
		download_data2, ok := userlib.DatastoreGet(useUUID)
		if !ok {
			return errors.New(strings.ToTitle("No file found 2x"))
		}

		verfied_error2 := userlib.DSVerify(vk, download_data2[256:], download_data2[0:256])
		if verfied_error2 != nil {
			return errors.New(strings.ToTitle("Error Verifying File"))
		}
		decrypted_data2 := userlib.SymDec(userlib.Argon2Key([]byte(userdata.Password+filename), userdata.salt, 32), download_data2[256:])
		if err != nil {
			return err
		}
		err = json.Unmarshal(decrypted_data2, &filePlacer)
		if err != nil {
			return err
		}

		var newContent LinkedList
		var newNode Node
		newNode.value = content
		newContent.head = &newNode
		newContent.tail = &newNode

		filePlacer.FileContent = newContent

		contentBytes, err := json.Marshal(filedata)
		if err != nil {
			return err
		}

		encrypted_contents := userlib.SymEnc(userlib.Argon2Key([]byte(userdata.Password+filename), userdata.salt, 32), userdata.salt, contentBytes)
		signature, err := userlib.DSSign(userdata.SignKey, encrypted_contents)
		if err != nil {
			return err
		}
		store := append(signature[:], encrypted_contents[:]...)

		userlib.DatastoreSet(useUUID, store)
	}
	return nil
}

func (userdata *User) AppendToFile(filename string, content []byte) error {
	key, err := uuid.FromBytes(userlib.Hash([]byte(filename + "-" + userdata.Username))[:16])
	if err != nil {
		return err
	}

	download_data, ok := userlib.DatastoreGet(key)
	if !ok {
		return errors.New(strings.ToTitle("This file does not exist"))
	}
	var filedata File
	var filePlacer File

	vk, ok := userlib.KeystoreGet("USER" + userdata.Username)
	if !ok {
		return errors.New(strings.ToTitle("Verify Key not Found"))
	}
	verfied_error := userlib.DSVerify(vk, download_data[256:], download_data[0:256])
	if verfied_error != nil {
		return errors.New(strings.ToTitle("Error Verifying File"))
	}
	decrypted_data := userlib.SymDec(userlib.Argon2Key([]byte(userdata.Password+filename), userdata.salt, 32), download_data[256:])
	if err != nil {
		return err
	}
	err = json.Unmarshal(decrypted_data, &filedata)

	useUUID := filedata.PlaceHolder
	download_data2, ok := userlib.DatastoreGet(useUUID)
	if !ok {
		return errors.New(strings.ToTitle("No file found 2x"))
	}

	verfied_error2 := userlib.DSVerify(vk, download_data2[256:], download_data2[0:256])
	if verfied_error2 != nil {
		return errors.New(strings.ToTitle("Error Verifying File"))
	}
	decrypted_data2 := userlib.SymDec(userlib.Argon2Key([]byte(userdata.Password+filename), userdata.salt, 32), download_data2[256:])
	if err != nil {
		return err
	}
	err = json.Unmarshal(decrypted_data2, &filePlacer)
	if err != nil {
		return err
	}

	filePlacer.FileContent.Insert(content)

	contentBytes, err := json.Marshal(filePlacer)
	if err != nil {
		return err
	}
	// RE-ENCRYPT EVERYTHING
	re_encrypted_contents := userlib.SymEnc(userlib.Argon2Key([]byte(userdata.Password+filename), userdata.salt, 32), userdata.salt, contentBytes)

	if err != nil {
		return err
	}
	// CREATE THE DIGITAL SIGNATURE
	signature, err := userlib.DSSign(userdata.SignKey, re_encrypted_contents)
	if err != nil {
		return err
	}
	// ATTATCH THE SIGNATURE TO THE BEGINNING OF THE ENCRYPTED STRUCT
	store := append(signature[:], re_encrypted_contents[:]...)
	userlib.DatastoreSet(useUUID, store)
	return nil
}

func (userdata *User) LoadFile(filename string) (content []byte, err error) {
	// check if 1 abstraction file even exists in userdata's name space
	storageKey, err := uuid.FromBytes(userlib.Hash([]byte(filename + "-" + userdata.Username))[:16])
	if err != nil {
		return nil, err
	}
	downloadData, ok := userlib.DatastoreGet(storageKey)
	if !ok {
		return nil, errors.New(strings.ToTitle("file not found"))
	}

	var filedata File
	var filePlacer File

	// DECRYPT THE FILE
	// decrypted_contents := userlib.SymDec(userdata.privateKey, downloadData[64:])
	// decrypt_download data QUESTION
	vk, ok := userlib.KeystoreGet("USER" + userdata.Username)
	if !ok {
		return nil, errors.New(strings.ToTitle("Verify Key not Found"))
	}
	verfied_error := userlib.DSVerify(vk, downloadData[256:], downloadData[0:256])
	if verfied_error != nil {
		return nil, errors.New(strings.ToTitle("Error Verifying File"))
	}
	decrypted_data := userlib.SymDec(userlib.Argon2Key([]byte(userdata.Password+filename), userdata.salt, 32), downloadData[256:])
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(decrypted_data, &filedata)

	useUUID := filedata.PlaceHolder
	download_data2, ok := userlib.DatastoreGet(useUUID)
	if !ok {
		return nil, errors.New(strings.ToTitle("No file found 2x"))
	}

	verfied_error2 := userlib.DSVerify(vk, download_data2[256:], download_data2[0:256])
	if verfied_error2 != nil {
		return nil, errors.New(strings.ToTitle("Error Verifying File"))
	}
	decrypted_data2 := userlib.SymDec(userlib.Argon2Key([]byte(userdata.Password+filename), userdata.salt, 32), download_data2[256:])
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(decrypted_data2, &filePlacer)

	if filePlacer.FileOwner != userdata.Username {
		// is this the right way of getting it?
		storageKey, err := uuid.FromBytes(userlib.Hash([]byte(filePlacer.FileOwner + "-" + userdata.Username))[:16])
		if err != nil {
			return nil, err
		}
		originalOwnerFile, ok := userlib.DatastoreGet(storageKey)
		if !ok {
			return nil, errors.New(strings.ToTitle("No file found 2x"))
		}
		decryptedInvite := userlib.SymDec(userlib.Argon2Key([]byte(userdata.Password+filename), userdata.salt, 32), originalOwnerFile[256:])
		var invite Intive
		json.Unmarshal(decryptedInvite, &invite)

		if invite.newSourceFile {
			var newUUID userlib.UUID = invite.newUUID
			// go to filename.content for my first abstraction
			download_data2, ok := userlib.DatastoreGet(newUUID)
			if !ok {
				return nil, errors.New(strings.ToTitle("No file found 2x"))
			}
			// WHICH KEY?
			verfied_error2 := userlib.DSVerify(vk, download_data2[256:], download_data2[0:256])
			if verfied_error2 != nil {
				return nil, errors.New(strings.ToTitle("Error Verifying File"))
			}
			decrypted_data2 := userlib.SymDec(userlib.Argon2Key([]byte(userdata.Password+filename), userdata.salt, 32), download_data2[256:])
			if err != nil {
				return nil, err
			}
			err = json.Unmarshal(decrypted_data2, &filePlacer)
			invite.newSourceFile = false
			filedata.PlaceHolder = newUUID
			// CHECK FOR ERRORS
			contentBytes, err := json.Marshal(filedata)
			re_encrypted_contents := userlib.SymEnc(userlib.Argon2Key([]byte(userdata.Password+filename), userdata.salt, 32), userdata.salt, contentBytes)

			if err != nil {
				return nil, err
			}
			// CREATE THE DIGITAL SIGNATURE
			signature, err := userlib.DSSign(userdata.SignKey, re_encrypted_contents)
			if err != nil {
				return nil, err
			}
			// ATTATCH THE SIGNATURE TO THE BEGINNING OF THE ENCRYPTED STRUCT
			store := append(signature[:], re_encrypted_contents[:]...)
			userlib.DatastoreSet(storageKey, store)
			// REENCRYPT AND RESIGN
		}
	}
	if err != nil {
		return nil, err
	}
	var loadedContents []byte
	ptr := filePlacer.FileContent.head
	for {
		if ptr == nil {
			break
		}
		new_contents := ptr.value
		loadedContents = append(loadedContents, new_contents[:]...)
		ptr = ptr.next
	}
	content = loadedContents
	// decrypt file contents
	return content, err
}

func (userdata *User) CreateInvitation(filename string, recipientUsername string) (
	invitationPtr uuid.UUID, err error) {
	// get recipient pub key
	recipientPublicKey, ok := userlib.KeystoreGet(recipientUsername)
	if !ok {
		return uuid.Nil, errors.New(strings.ToTitle("pub key not found"))
	}
	// create instance of invite
	var invitedata Invite
	// set values
	invitedata.FilePath = filename + '-' + userdata.Username // hash the file path

	storageKey, err := uuid.FromBytes(userlib.Hash([]byte(invitedata.FilePath))[:16])                                                 // use hashed version of filepath as uuid?
	invitedata.PublicEncSymmKey = PKEEnc(recipientPublicKey, SymDec(Argon2Key(userdata.Password, salt, keyLen), userdata.PrivateKey)) // load symm key of file from
	contentBytes, err := json.Marshal(invitedata)

	userlib.DatastoreSet(storageKey, contentBytes)

	return &invitedata, nil
}

func (userdata *User) AcceptInvitation(senderUsername string, invitationPtr uuid.UUID, filename string) error {
	// call store file which in content argument calls load file in the owner's name space
	// store file uses uuid made as 'userdata.username-filename'
	// storefile('recipient given name', Alice.loadfile('abc'))

	// fetch my private key
	// un marshal invite json
	// use my private key on invitedata.PublicEncSymmKey to get hash(sender's pvt key - filename)
	// use storefile('my given name', loadfile)
	err, encryptedContentRecieved := userlib.DatastoreGet(invitationPtr)
	//now decrypt encryptedContentRecieved
	// call store file
	return nil
}

func (userdata *User) RevokeAccess(filename string, recipientUsername string) error {
	// create new symm key for file
	// encrypt file with new symmetric key
	// remove recipientUsername from file access table
	// encrypt new symmetric key with every user remaining on access table's public key
	//
	return nil
}

// func (l *List) InsertAt(pos int, value byte[]) {
// 	// create a new node
// 	newNode := Node{}
// 	newNode.value = value
// 	// validate the position
// 	if pos < 0 {
// 		return
// 	}
// 	if pos == 0 {
// 		l.head = &newNode
// 		l.len++
// 		return
// 	}
// 	if pos > l.len {
// 		return
// 	}

// 	ptr := l.tail

// 	n := l.GetAt(pos)
// 	newNode.next = n
// 	prevNode := l.GetAt(pos - 1)
// 	prevNode.next = &newNode
// 	l.len++
// }

// Insert inserts new node at the end of  from linked list
func (l *LinkedList) Insert(val []byte) {
	n := Node{}
	n.value = val
	l.tail.next = &n
	l.tail = l.tail.next
}
