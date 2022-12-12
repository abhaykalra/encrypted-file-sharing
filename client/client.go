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
	PrivateKey userlib.PKEDecKey
	Salt       []byte
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
	LocationOfFile userlib.UUID
	KeyOfFile      []byte
	SaltOfFile     []byte
	NewSourceFile  bool
}

type File struct {
	SharedWith   map[string][]string // file owner - file name (given by owner)
	FileOwner    string
	FileContent  LinkedList
	PlaceHolder  userlib.UUID
	OwnSecond    bool
	KeyOfSecond  []byte
	SaltOfSecond []byte
}

type Node struct {
	Next *Node
	Val  []byte
}

type LinkedList struct {
	Head *Node
	Tail *Node
}

// storageKey, err := uuid.FromBytes(userlib.Hash([]byte(filename + userdata.Username))[:16])

func InitUser(username string, password string) (userdataptr *User, err error) {
	if username == "" {
		return nil, errors.New(strings.ToTitle("CAN'T HAVE A BLANK USERNAME"))
	}
	checkKey, err := uuid.FromBytes(userlib.Hash([]byte(username))[:16])
	_, ok := userlib.DatastoreGet(checkKey)
	if ok {
		return nil, errors.New(strings.ToTitle("USER ALREADY EXISTS"))
	}
	var userdata User            // create instance of a user
	userdata.Username = username // set username based on input
	userdata.Password = password // set password based on input and encrypt using ____ scheme??
	// look into how salts can be used here
	var salt []byte = userlib.Hash([]byte(username + password))[:16]
	userdata.Salt = salt
	var ENCpk userlib.PKEEncKey                                        //make public key
	var DECsk userlib.PKEDecKey                                        //make private key
	ENCpk, DECsk, _ = userlib.PKEKeyGen()                              // generate both keys
	var rootKey []byte = userlib.Argon2Key([]byte(password), salt, 16) //generate root key

	// var skEncrypted []byte = SymEnc(rootKey , iv []byte, sk) // encrypt sk with root key
	userdata.PrivateKey = DECsk // store private key pkdf encryped in datastore
	userlib.KeystoreSet(username, ENCpk)

	storageKey, err := uuid.FromBytes(userlib.Hash([]byte(username))[:16])

	// CREATE DIGITAL SIGNATURE METHODS
	sk, vk, err := userlib.DSKeyGen()
	if err != nil {
		return nil, errors.New(strings.ToTitle("173"))
	}
	userdata.SignKey = sk

	userlib.KeystoreSet("USER"+username, vk) // store users public key for signature verification in keystore

	if err != nil {
		return nil, err
	}

	contentBytes, err := json.Marshal(userdata)

	// ENCRYPT THE STRUCT
	encryptedStruct := userlib.SymEnc(rootKey, salt, contentBytes)
	// CREATE THE DIGITAL SIGNATURE
	signature, err := userlib.DSSign(sk, encryptedStruct)

	if err != nil {
		return nil, errors.New(strings.ToTitle("186"))
	}

	// ATTATCH THE SIGNATURE TO THE BEGINNING OF THE ENCRYPTED STRUCT
	store := append(signature[:], encryptedStruct[:]...)
	userlib.DatastoreSet(storageKey, store)

	// store users public key in keystore against their username as a key

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
		return nil, errors.New(strings.ToTitle("Error Verifying Userstruct 220"))
	}

	// DERIVE THE DECRYPTION METHOD TO DECRYPT THE USER STRUCT
	var salt []byte = userlib.Hash(([]byte(username + password)))[:16]
	var rootKey []byte = userlib.Argon2Key([]byte(password), salt, 16) //generate root key

	// DECRYPT THE USER STRUCT
	dataJSON = userlib.SymDec(rootKey, dataJSON[256:])
	userdataptr = &userdata
	err = json.Unmarshal(dataJSON, userdataptr) //You can derive two keys from one key, then use those two keys to encrypt and MAC.
	if err != nil {
		return nil, errors.New(strings.ToTitle("Error Unmarshling 230"))
	}
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
		filedata.OwnSecond = true
		filedata.FileOwner = userdata.Username

		// Start storing the real file
		fileContents := LinkedList{}
		fileContents.Head = NewNode(content, nil)

		// fileContents.Insert(content)
		x := map[string][]string{
			userdata.Username: {}}
		filePlacer.FileContent = fileContents
		filePlacer.SharedWith = x
		filePlacer.FileOwner = userdata.Username

		content_second_file, err := json.Marshal(filePlacer)
		if err != nil {
			return err
		}
		// Encrypt and store the file with the actual contents
		passGenKey := userlib.Argon2Key([]byte(userdata.Password+filename), userdata.Salt, 32) // returns []bytre
		encrypt_key := passGenKey[:16]
		mac_key := passGenKey[16:32]
		encrypted_second_file := userlib.SymEnc(encrypt_key, userdata.Salt, content_second_file)
		mac_second_file, err := userlib.HMACEval(mac_key, encrypted_second_file)
		if err != nil {
			return err
		}
		maced_second_file := append(mac_second_file[:], encrypted_second_file[:]...)
		userlib.DatastoreSet(randomUUID, maced_second_file)
		// ADD THESE VALUES TO MY FIRST FILE
		filedata.KeyOfSecond = passGenKey
		filedata.SaltOfSecond = userdata.Salt

		//ENCRYPT FIRST FILE
		contentFirst, err := json.Marshal(filedata)
		if err != nil {
			return err
		}
		encrypted_contents := userlib.SymEnc(userlib.Argon2Key([]byte(userdata.Password+filename+userdata.Username), userdata.Salt, 16), userdata.Salt, contentFirst)
		signature, err := userlib.DSSign(userdata.SignKey, encrypted_contents)
		if err != nil {
			return errors.New(strings.ToTitle("297"))
		}
		store := append(signature[:], encrypted_contents[:]...)
		// Store the fake file
		userlib.DatastoreSet(key, store)

	} else {
		// decrypt_download data QUESTION
		vk, ok := userlib.KeystoreGet("USER" + userdata.Username)
		if !ok {
			return errors.New(strings.ToTitle("Verify Key not Found"))
		}
		verfied_error := userlib.DSVerify(vk, download_data[256:], download_data[0:256])
		if verfied_error != nil {
			return errors.New(strings.ToTitle("Error Verifying File 311"))
		}
		decrypted_data := userlib.SymDec(userlib.Argon2Key([]byte(userdata.Password+filename+userdata.Username), userdata.Salt, 16), download_data[256:])

		err = json.Unmarshal(decrypted_data, &filedata)
		if err != nil {
			return errors.New(strings.ToTitle("Error Unmarshling 318"))
		}
		var decryptkey []byte
		var decryptSalt []byte
		var macKey []byte
		if !filedata.OwnSecond {
			decryptkey = filedata.KeyOfSecond[:16]
			macKey = filedata.KeyOfSecond[16:]
			decryptSalt = filedata.SaltOfSecond
		} else {
			decryptSalt = userdata.Salt
			decryptkey = userlib.Argon2Key([]byte(userdata.Password+filename), userdata.Salt, 32)[:16]
			macKey = userlib.Argon2Key([]byte(userdata.Password+filename), userdata.Salt, 32)[16:]

		}
		useUUID := filedata.PlaceHolder
		if !filedata.OwnSecond {
			//IOF DONT OWN SECOND
			// is this the right way of getting it?

			invite := userdata.helperGetToInvite(filedata.FileOwner)

			// OLD PASSWORD FROM THIS USERS FIRST FILE STAYS RELEVANT
			if invite.NewSourceFile {
				useUUID = invite.LocationOfFile

				userdata.updateInvitation(filedata.FileOwner)
				filedata.PlaceHolder = useUUID
				// *******************************
				// *******************************

				// ONCE WE MAKE CHANGES TO THE INVITE (CGHNAGE THE NEWSOURCFILE TAG BACK TO FALSE)
				// WE NEED TO ENCRYPT USING OUR PUBLIC KEY AND THEN

				// CHECK FOR ERRORS

				// ATTATCH THE SIGNATURE TO THE BEGINNING OF THE ENCRYPTED STRUCT

				contentBytes, err := json.Marshal(filedata)
				if err != nil {
					return errors.New(strings.ToTitle("Error Marshal line 826"))
				}
				re_encrypted_contents := userlib.SymEnc(userlib.Argon2Key([]byte(userdata.Password+filename+userdata.Username), userdata.Salt, 16), userdata.Salt, contentBytes)

				if err != nil {
					return errors.New(strings.ToTitle("Error encrypt line 831"))
				}
				// CREATE THE DIGITAL SIGNATURE
				signature, err := userlib.DSSign(userdata.SignKey, re_encrypted_contents)
				if err != nil {
					return errors.New(strings.ToTitle("Error signing line 367"))
				}
				// ATTATCH THE SIGNATURE TO THE BEGINNING OF THE ENCRYPTED STRUCT
				store := append(signature[:], re_encrypted_contents[:]...)
				userlib.DatastoreSet(key, store)
				// REENCRYPT AND RESIGN
			}

		}
		download_data2, ok := userlib.DatastoreGet(useUUID)
		if !ok {
			return errors.New(strings.ToTitle("No file found 2x 379"))
		}

		mac_of_data, err := userlib.HMACEval(macKey, download_data2[64:])
		if err != nil {
			return errors.New(strings.ToTitle("Error unmacing"))
		}
		ok = userlib.HMACEqual(mac_of_data, download_data2[:64])
		if !ok {
			return errors.New(strings.ToTitle("The MAC doesn't match for the main file"))
		}
		// Decrypt the second layer of abstraction
		decrypted_data2 := userlib.SymDec(decryptkey, download_data2[64:])

		err = json.Unmarshal(decrypted_data2, &filePlacer)
		if err != nil {
			return errors.New(strings.ToTitle("Error UnMarshalling 682"))
		}

		// var newContent LinkedList

		fileContents := LinkedList{}
		fileContents.Head = NewNode(content, nil)

		// fileContents.Insert(content)

		filePlacer.FileContent = fileContents

		// filePlacer.FileContent = newContent

		contentSecond, err := json.Marshal(filePlacer)
		if err != nil {
			return err
		}

		encrypted_contents_second := userlib.SymEnc(decryptkey, decryptSalt, contentSecond)
		mac_for_file, err := userlib.HMACEval(macKey, encrypted_contents_second)
		if err != nil {
			return err
		}
		maced_second_file := append(mac_for_file[:], encrypted_contents_second[:]...)
		userlib.DatastoreSet(useUUID, maced_second_file)

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
		return errors.New(strings.ToTitle("Error Verifying File 444"))
	}
	decrypted_data := userlib.SymDec(userlib.Argon2Key([]byte(userdata.Password+filename+userdata.Username), userdata.Salt, 16), download_data[256:])

	err = json.Unmarshal(decrypted_data, &filedata)
	if err != nil {
		return errors.New(strings.ToTitle("Error Unmarshling 492"))
	}

	var decryptkey []byte
	var decryptSalt []byte
	var macKey []byte
	if !filedata.OwnSecond {
		decryptkey = filedata.KeyOfSecond[:16]
		macKey = filedata.KeyOfSecond[16:]
		decryptSalt = filedata.SaltOfSecond
	} else {
		decryptSalt = userdata.Salt
		decryptkey = userlib.Argon2Key([]byte(userdata.Password+filename), userdata.Salt, 32)[:16]
		macKey = userlib.Argon2Key([]byte(userdata.Password+filename), userdata.Salt, 32)[16:]
	}

	// HERES WHERE WE CHECK THE INVITE
	var useUUID userlib.UUID = filedata.PlaceHolder
	// initialised useuuid as original uuid from 1st degree file

	if !filedata.OwnSecond {
		invite1 := userdata.helperGetToInvite(filedata.FileOwner)
		// GET THE INVITE
		// OLD PASSWORD FROM THIS USERS FIRST FILE STAYS RELEVANT
		if invite1.NewSourceFile {
			useUUID = invite1.LocationOfFile
			// go to filename.content for my first abstraction

			userdata.updateInvitation(filedata.FileOwner)
			filedata.PlaceHolder = useUUID
			// *******************************
			// *******************************

			// ONCE WE MAKE CHANGES TO THE INVITE (CGHNAGE THE NEWSOURCFILE TAG BACK TO FALSE)
			// WE NEED TO ENCRYPT USING OUR PUBLIC KEY AND THEN

			// PUT THE INVITE BACK ONTO DATASTORE

			// NOW WE FIX THE PLACEHOLDER FILE
			contentBytes, err := json.Marshal(filedata)
			if err != nil {
				return err
			}
			re_encrypted_contents := userlib.SymEnc(userlib.Argon2Key([]byte(userdata.Password+filename+userdata.Username), userdata.Salt, 16), userdata.Salt, contentBytes)

			if err != nil {
				return err
			}
			// CREATE THE DIGITAL SIGNATURE
			signature, err := userlib.DSSign(userdata.SignKey, re_encrypted_contents)
			if err != nil {
				return errors.New(strings.ToTitle("user not found 501"))
			}
			// ATTATCH THE SIGNATURE TO THE BEGINNING OF THE ENCRYPTED STRUCT
			store := append(signature[:], re_encrypted_contents[:]...)
			userlib.DatastoreSet(key, store)

			//NEED TO ACTUALLY DO APPENDS
		} // if no new source file also invite and filedata open
		// all we did was open file placer in the case if no new source file
		//NO NEED TO CLOSE INVITE OR FIRST FILE SINCE WE DIDNT CHANGE ANYTHING

	}

	// I OWN SECOND FILE
	// FIRST FILE IS OPEN
	//USEUUID IS ORIGINAL
	download_data2, ok := userlib.DatastoreGet(useUUID)
	if !ok {
		return errors.New(strings.ToTitle("No file found 2x 520"))
	}

	mac_of_data, err := userlib.HMACEval(macKey, download_data2[64:])
	if err != nil {
		return errors.New(strings.ToTitle("Error unmacing"))
	}
	ok = userlib.HMACEqual(mac_of_data, download_data2[:64])
	if !ok {
		return errors.New(strings.ToTitle("The MAC doesn't match for the main file"))
	}
	// Decrypt the second layer of abstraction
	decrypted_data2 := userlib.SymDec(decryptkey, download_data2[64:])

	err = json.Unmarshal(decrypted_data2, &filePlacer)
	if err != nil {
		return errors.New(strings.ToTitle("Error UnMarshalling 682"))
	}
	// FINALLY APPEND TO FILE BUT THIS HAS ONLY HAPPENED IF I OWNED 2NF FILE
	//filePlacer.FileContent.(content)
	AddNodeAtEnd(filePlacer.FileContent.Head, content)

	contentBytes, err := json.Marshal(filePlacer)
	if err != nil {
		return err
	}
	// RE-ENCRYPT EVERYTHING
	re_encrypted_contents := userlib.SymEnc(decryptkey, decryptSalt, contentBytes)
	mac_for_file, err := userlib.HMACEval(macKey, re_encrypted_contents)
	if err != nil {
		return err
	}
	maced_second_file := append(mac_for_file[:], re_encrypted_contents[:]...)
	userlib.DatastoreSet(useUUID, maced_second_file)

	return nil
}

func (userdata *User) LoadFile(filename string) (content []byte, err error) {
	// check if 1 abstraction file even exists in userdata's name space
	storageKey, err := uuid.FromBytes(userlib.Hash([]byte(filename + "-" + userdata.Username))[:16])
	if err != nil {
		return nil, errors.New(strings.ToTitle("file not found"))
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
		return nil, errors.New(strings.ToTitle("Error Verifying File 580"))
	}
	decrypted_data := userlib.SymDec(userlib.Argon2Key([]byte(userdata.Password+filename+userdata.Username), userdata.Salt, 16), downloadData[256:])

	err = json.Unmarshal(decrypted_data, &filedata)
	if err != nil {
		return nil, errors.New(strings.ToTitle("Error Unmarshling 712"))
	}

	var decryptkey []byte
	var macKey []byte
	if !filedata.OwnSecond {
		decryptkey = filedata.KeyOfSecond[:16]
		macKey = filedata.KeyOfSecond[16:]

	} else {
		decryptkey = userlib.Argon2Key([]byte(userdata.Password+filename), userdata.Salt, 32)[:16]
		macKey = userlib.Argon2Key([]byte(userdata.Password+filename), userdata.Salt, 32)[16:]
	}

	var newUUID userlib.UUID = filedata.PlaceHolder
	// VERFYING THE INVITE, WE ARE GETTING THE FILE OWNER KEY
	if !filedata.OwnSecond {

		// is this the right way of getting it?
		invite := userdata.helperGetToInvite(filedata.FileOwner)
		// OLD PASSWORD FROM THIS USERS FIRST FILE STAYS RELEVANT
		if invite.NewSourceFile {
			newUUID = invite.LocationOfFile

			userdata.updateInvitation(filedata.FileOwner)
			filedata.PlaceHolder = newUUID
			// *******************************
			// *******************************

			// ONCE WE MAKE CHANGES TO THE INVITE (CGHNAGE THE NEWSOURCFILE TAG BACK TO FALSE)
			// WE NEED TO ENCRYPT USING OUR PUBLIC KEY AND THEN

			// CHECK FOR ERRORS

			// ATTATCH THE SIGNATURE TO THE BEGINNING OF THE ENCRYPTED STRUCT

			contentBytes, err := json.Marshal(filedata)
			if err != nil {
				return nil, errors.New(strings.ToTitle("Error Marshal line 826"))
			}
			re_encrypted_contents := userlib.SymEnc(userlib.Argon2Key([]byte(userdata.Password+filename+userdata.Username), userdata.Salt, 16), userdata.Salt, contentBytes)

			if err != nil {
				return nil, errors.New(strings.ToTitle("Error encrypt line 831"))
			}
			// CREATE THE DIGITAL SIGNATURE
			signature, err := userlib.DSSign(userdata.SignKey, re_encrypted_contents)
			if err != nil {
				return nil, errors.New(strings.ToTitle("Error signing line 634"))
			}
			// ATTATCH THE SIGNATURE TO THE BEGINNING OF THE ENCRYPTED STRUCT
			store := append(signature[:], re_encrypted_contents[:]...)
			userlib.DatastoreSet(storageKey, store)
			// REENCRYPT AND RESIGN

		}
	}

	download_data2, ok := userlib.DatastoreGet(newUUID)
	userlib.DebugMsg("641 output: %v", newUUID.String())
	if !ok {
		return nil, errors.New(strings.ToTitle("No file found 2x 648"))
	}

	mac_of_data, err := userlib.HMACEval(macKey, download_data2[64:])
	if err != nil {
		return nil, errors.New(strings.ToTitle("Error unmacing"))
	}
	ok = userlib.HMACEqual(mac_of_data, download_data2[:64])
	if !ok {
		return nil, errors.New(strings.ToTitle("The MAC doesn't match for the main file"))
	}
	// Decrypt the second layer of abstraction
	decrypted_data2 := userlib.SymDec(decryptkey, download_data2[64:])

	err = json.Unmarshal(decrypted_data2, &filePlacer)
	if err != nil {
		return nil, errors.New(strings.ToTitle("Error Marshalling 864"))
	}
	// EVERYTHING FINE SO FAR
	var loadedContents []byte
	ptr := filePlacer.FileContent.Head
	//var i int = 0
	for ptr != nil {
		new_contents := ptr.Val
		loadedContents = append(loadedContents, new_contents[:]...)
		ptr = ptr.Next
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
		return uuid.Nil, errors.New(strings.ToTitle("public key not found"))
	}
	key, err := uuid.FromBytes(userlib.Hash([]byte(filename + "-" + userdata.Username))[:16])
	if err != nil {
		return uuid.Nil, errors.New(strings.ToTitle("This key cant be made"))
	}

	download_data, ok := userlib.DatastoreGet(key)
	if !ok {
		return uuid.Nil, errors.New(strings.ToTitle("This file does not exist"))
	}
	var filedata File
	var filePlacer File
	userlib.DebugMsg("initial arg to keystore get: %v", "USER"+userdata.Username)
	vk, ok := userlib.KeystoreGet("USER" + userdata.Username)
	if !ok {
		return uuid.Nil, errors.New(strings.ToTitle("Verify Key not Found 702"))
	}
	verfied_error := userlib.DSVerify(vk, download_data[256:], download_data[0:256])
	if verfied_error != nil {
		return uuid.Nil, errors.New(strings.ToTitle("Error Verifying File 706"))
	}
	decrypted_data := userlib.SymDec(userlib.Argon2Key([]byte(userdata.Password+filename+userdata.Username), userdata.Salt, 16), download_data[256:])

	err = json.Unmarshal(decrypted_data, &filedata)
	if err != nil {
		return uuid.Nil, errors.New(strings.ToTitle("Error Unmarshling 712"))
	}

	var decryptkey []byte
	var decryptSalt []byte
	var macKey []byte
	if !filedata.OwnSecond {
		decryptkey = filedata.KeyOfSecond[:16]
		macKey = filedata.KeyOfSecond[16:]
		decryptSalt = filedata.SaltOfSecond
	} else {
		decryptSalt = userdata.Salt
		decryptkey = userlib.Argon2Key([]byte(userdata.Password+filename), userdata.Salt, 32)[:16]
		macKey = userlib.Argon2Key([]byte(userdata.Password+filename), userdata.Salt, 32)[16:]
	}
	userlib.DebugMsg("NO ERRORS THROUGH 727: %v", "USER"+userdata.Username)

	// HERES WHERE WE CHECK THE INVITE
	var useUUID userlib.UUID = filedata.PlaceHolder
	// initialised useuuid as original uuid from 1st degree file
	if !filedata.OwnSecond {
		// GET THE INVITE
		invite := userdata.helperGetToInvite(filedata.FileOwner)
		// OLD PASSWORD FROM THIS USERS FIRST FILE STAYS RELEVANT
		if invite.NewSourceFile {
			useUUID = invite.LocationOfFile
			// go to filename.content for my first abstraction

			userdata.updateInvitation(filedata.FileOwner)
			filedata.PlaceHolder = useUUID
			// *******************************
			// *******************************

			// ONCE WE MAKE CHANGES TO THE INVITE (CGHNAGE THE NEWSOURCFILE TAG BACK TO FALSE)
			// WE NEED TO ENCRYPT USING OUR PUBLIC KEY AND THEN

			// PUT THE INVITE BACK ONTO DATASTORE

			// NOW WE FIX THE PLACEHOLDER FILE

			//NEED TO ACTUALLY DO APPENDS
		} // if no new source file also invite and filedata open
		// all we did was open file placer in the case if no new source file
		//NO NEED TO CLOSE INVITE OR FIRST FILE SINCE WE DIDNT CHANGE ANYTHING
	}

	// I OWN SECOND FILE
	// FIRST FILE IS OPEN
	// USEUUID IS ORIGINAL
	download_data2, ok := userlib.DatastoreGet(useUUID)
	if !ok {
		return uuid.Nil, errors.New(strings.ToTitle("No file found 2x 765"))
	}

	mac_of_data, err := userlib.HMACEval(macKey, download_data2[64:])
	if err != nil {
		return uuid.Nil, errors.New(strings.ToTitle("Error unmacing"))
	}
	ok = userlib.HMACEqual(mac_of_data, download_data2[:64])
	if !ok {
		return uuid.Nil, errors.New(strings.ToTitle("The MAC doesn't match for the main file"))
	}
	// Decrypt the second layer of abstraction
	userlib.DebugMsg("NO ERRORS THROUGH 775: %v", "USER"+userdata.Username)
	decrypted_data2 := userlib.SymDec(decryptkey, download_data2[64:])

	err = json.Unmarshal(decrypted_data2, &filePlacer)
	if err != nil {
		return uuid.Nil, errors.New(strings.ToTitle("Error UnMarshalling 682"))
	}

	var strSlice []string
	filePlacer.SharedWith[userdata.Username] = append(filePlacer.SharedWith[userdata.Username], recipientUsername)
	filePlacer.SharedWith[recipientUsername] = strSlice
	if userdata.Username != filedata.FileOwner {
		filePlacer.SharedWith[filedata.FileOwner] = append(filePlacer.SharedWith[filedata.FileOwner], recipientUsername)

	}

	// FINALLY APPEND TO FILE BUT THIS HAS ONLY HAPPENED IF I OWNED 2NF FILE

	contentBytes, err := json.Marshal(filePlacer)
	if err != nil {
		return uuid.Nil, errors.New(strings.ToTitle("Error creating marshal line 1060"))
	}
	// RE-ENCRYPT EVERYTHING
	re_encrypted_contents := userlib.SymEnc(decryptkey, decryptSalt, contentBytes)
	mac_for_file, err := userlib.HMACEval(macKey, re_encrypted_contents)
	if err != nil {
		return uuid.Nil, errors.New(strings.ToTitle("Error creating mac"))
	}
	maced_second_file := append(mac_for_file[:], re_encrypted_contents[:]...)
	userlib.DatastoreSet(useUUID, maced_second_file)

	// CREATE ACTUAL INVITE #########################################################################

	var invitedata Invite
	// get random file uuid
	// add case when i dont own secondfile
	// either get them from my fildata (IF THOSE VLAUES WERE SET IN STORE FILE)
	invitedata.LocationOfFile = useUUID
	invitedata.KeyOfFile = filedata.KeyOfSecond
	invitedata.SaltOfFile = filedata.SaltOfSecond
	randomUUIDInvitehash := userlib.Hash([]byte("-" + userdata.Username + "-" + recipientUsername)) // use double --
	randomUUIDInvite, err := uuid.FromBytes(randomUUIDInvitehash[:16])
	if err != nil {
		return uuid.Nil, errors.New(strings.ToTitle("Error creating uuid "))
	}
	// *** NOW LOAD FILE SHOULD HANDLE THE FETCHING->CHECKING MAC->DECRYPTING->UNMARSHALING
	// create uuid to put invite struct at

	// marshal invite struct
	jsonInvite, err := json.Marshal(invitedata)
	if err != nil {
		return uuid.Nil, errors.New(strings.ToTitle("Error marshling line 984"))
	}
	// encrypt json of invite struct w recipient symm key

	encKeyInviteAdjusted := userlib.Argon2Key([]byte(recipientUsername+userdata.Username), userdata.Salt, 16)

	saltInviteAdjusted := userdata.Salt
	// symm enc w random gen key
	encrypted_invite := userlib.SymEnc(encKeyInviteAdjusted, saltInviteAdjusted, jsonInvite)

	//userlib.DebugMsg("NO ERRORS THROUGH 775: %v", "USER"+userdata.Username)
	//userlib.DebugMsg("type of key in question: %v", reflect.TypeOf(userdata.SignKey))
	//userlib.DebugMsg("THIS IS THE KEY: %v", userdata.SignKey)
	signature, err := userlib.DSSign(userdata.SignKey, encrypted_invite)
	if err != nil {
		return uuid.Nil, errors.New(strings.ToTitle("836"))
	}

	// ATTATCH THE SIGNATURE TO THE BEGINNING OF THE ENCRYPTED STRUCT
	store := append(signature[:], encrypted_invite[:]...)
	// put onto data store
	userlib.DatastoreSet(randomUUIDInvite, store)

	storageKeyForInviteUUID, err := uuid.FromBytes(userlib.Hash([]byte(userdata.Username + "-" + recipientUsername))[:16])
	if err != nil {
		return uuid.Nil, errors.New(strings.ToTitle("Error creating UUID"))
	}
	// first 16 is key next 16 is salt remaining is randomuuid

	tempappend := append(encKeyInviteAdjusted[:], saltInviteAdjusted[:]...)
	information_for_invite := append(tempappend, randomUUIDInvitehash[:16]...)
	information_for_invite_enc, err := userlib.PKEEnc(recipientPublicKey, information_for_invite)
	if err != nil {
		return uuid.Nil, errors.New(strings.ToTitle("Error Encrypting 997"))
	}
	signatureSec, err := userlib.DSSign(userdata.SignKey, information_for_invite_enc)
	if err != nil {
		return uuid.Nil, errors.New(strings.ToTitle("858"))
	}

	// ATTATCH THE SIGNATURE TO THE BEGINNING OF THE ENCRYPTED STRUCT
	storeSec := append(signatureSec[:], information_for_invite_enc[:]...)
	// put onto data store
	userlib.DatastoreSet(storageKeyForInviteUUID, storeSec)

	contentBytes, err = json.Marshal(filedata)
	if err != nil {
		return uuid.Nil, errors.New(strings.ToTitle("error marshalling"))
	}
	re_encrypted_contents = userlib.SymEnc(userlib.Argon2Key([]byte(userdata.Password+filename+userdata.Username), userdata.Salt, 16), userdata.Salt, contentBytes)

	if err != nil {
		return uuid.Nil, errors.New(strings.ToTitle("error encrypting 1017"))
	}
	// CREATE THE DIGITAL SIGNATURE
	signature, err = userlib.DSSign(userdata.SignKey, re_encrypted_contents)
	if err != nil {
		return uuid.Nil, errors.New(strings.ToTitle("error signing 878"))
	}
	// ATTATCH THE SIGNATURE TO THE BEGINNING OF THE ENCRYPTED STRUCT
	store = append(signature[:], re_encrypted_contents[:]...)
	userlib.DatastoreSet(key, store)

	return storageKeyForInviteUUID, nil // CHECK W SOMEONE!!!!!!!!!!!

}

func (userdata *User) AcceptInvitation(senderUsername string, invitationPtr uuid.UUID, filename string) error {
	// get priavtekey
	privateKey := userdata.PrivateKey
	// datastore get invitation
	inviteInfoEnc, ok := userlib.DatastoreGet(invitationPtr)
	if !ok {
		return errors.New(strings.ToTitle("No invite found"))
	}
	// check signature
	senderSig, ok := userlib.KeystoreGet("USER" + senderUsername)
	if !ok {
		return errors.New(strings.ToTitle("Verify Key not Found"))
	}
	verfied_error := userlib.DSVerify(senderSig, inviteInfoEnc[256:], inviteInfoEnc[0:256])
	if verfied_error != nil {
		return errors.New(strings.ToTitle("Error Verifying File 903"))
	}
	//decrypt using my private key
	inviteInfo, err := userlib.PKEDec(privateKey, inviteInfoEnc[256:])
	if err != nil {
		return errors.New(strings.ToTitle("Error PKEDec"))
	}
	var keyofInvite []byte
	//var saltOfInvite []byte
	var uuidOfInvite []byte

	keyofInvite = inviteInfo[:16]
	//saltOfInvite = inviteInfo[16:32]
	uuidOfInvite = inviteInfo[32:]

	uuidToFetch, err := uuid.FromBytes(uuidOfInvite)
	if err != nil {
		return errors.New(strings.ToTitle("Error creating uuid "))
	}
	invitedataEnc, ok := userlib.DatastoreGet(uuidToFetch)
	if !ok {
		return errors.New(strings.ToTitle("No invite found"))
	}
	senderSig2, ok := userlib.KeystoreGet("USER" + senderUsername)
	if !ok {
		return errors.New(strings.ToTitle("Verify Key not Found"))
	}
	verfied_error2 := userlib.DSVerify(senderSig2, invitedataEnc[256:], invitedataEnc[0:256])
	if verfied_error2 != nil {
		return errors.New(strings.ToTitle("Error Verifying File 932"))
	}
	inviteContent2 := userlib.SymDec(keyofInvite, invitedataEnc[256:])
	if err != nil {
		panic(errors.New("Error getting the string+key for the invite in the InviteHelper " + err.Error()))
	}
	//unmarshal
	var invitePlacer Invite
	err = json.Unmarshal(inviteContent2, &invitePlacer)
	if err != nil {
		return errors.New(strings.ToTitle("Error Unmarshling line 1029"))
	}
	// get the file random uuid
	fileUUID := invitePlacer.LocationOfFile
	fileSalt := invitePlacer.SaltOfFile
	fileKeyy := invitePlacer.KeyOfFile
	// get file content ?
	// store file in my namespace
	// create a new first degree abstraction (fake file)
	key, err := uuid.FromBytes(userlib.Hash([]byte(filename + "-" + userdata.Username))[:16])
	if err != nil {
		return err
	}

	var filedata File

	filedata.PlaceHolder = fileUUID
	filedata.OwnSecond = false
	filedata.KeyOfSecond = fileKeyy
	filedata.SaltOfSecond = fileSalt
	filedata.FileOwner = senderUsername
	contentBytes, err := json.Marshal(filedata)
	if err != nil {
		return errors.New(strings.ToTitle("Error marshling (line 1053)"))
	}

	encrypted_contents := userlib.SymEnc(userlib.Argon2Key([]byte(userdata.Password+filename+userdata.Username), userdata.Salt, 16), userdata.Salt, contentBytes)

	signature, err := userlib.DSSign(userdata.SignKey, encrypted_contents)
	if err != nil {
		return errors.New(strings.ToTitle("972"))
	}
	store := append(signature[:], encrypted_contents[:]...)
	// Store the fake file
	userlib.DatastoreSet(key, store)

	// check if second even exists still
	_, ok = userlib.DatastoreGet(fileUUID)
	if !ok {
		return errors.New(strings.ToTitle("No invite found"))
	}
	//
	return nil
}

func (userdata *User) RevokeAccess(filename string, recipientUsername string) error {
	// **REMEMBER only owner can call this
	// check if user owns second degree abstraction
	key, err := uuid.FromBytes(userlib.Hash([]byte(filename + "-" + userdata.Username))[:16])
	if err != nil {
		return errors.New(strings.ToTitle("Error UUID"))
	}
	fileEncryptedandSigned, ok := userlib.DatastoreGet(key)
	if !ok {
		return errors.New(strings.ToTitle("Datastore Error"))
	}
	var filedata File
	var filePlacer File
	vk, ok := userlib.KeystoreGet("USER" + userdata.Username)
	if !ok {
		return errors.New(strings.ToTitle("Verify Key not Found"))
	}
	verfied_error := userlib.DSVerify(vk, fileEncryptedandSigned[256:], fileEncryptedandSigned[0:256])
	if verfied_error != nil {
		return errors.New(strings.ToTitle("Error Verifying File 1000"))
	}

	fileOpen := userlib.SymDec(userlib.Argon2Key([]byte(userdata.Password+filename+userdata.Username), userdata.Salt, 16), fileEncryptedandSigned[256:])
	err = json.Unmarshal(fileOpen, &filedata)
	if err != nil {
		return errors.New(strings.ToTitle("Error Unmarshling (lin 1094)"))
	}
	amIowner := filedata.OwnSecond
	addressOfSecondAbstraction := filedata.PlaceHolder

	var decryptkey []byte
	var decryptSalt []byte
	var macKey []byte
	if !filedata.OwnSecond {
		decryptkey = filedata.KeyOfSecond[:16]
		macKey = filedata.KeyOfSecond[16:]
		decryptSalt = filedata.SaltOfSecond
	} else {
		decryptSalt = userdata.Salt
		decryptkey = userlib.Argon2Key([]byte(userdata.Password+filename), userdata.Salt, 32)[:16]
		macKey = userlib.Argon2Key([]byte(userdata.Password+filename), userdata.Salt, 32)[16:]
	}

	if amIowner {
		mainFile, ok := userlib.DatastoreGet(addressOfSecondAbstraction)
		if !ok {
			return errors.New(strings.ToTitle("Datastore Error"))
		}
		// check sign
		mac_of_data, err := userlib.HMACEval(macKey, mainFile[64:])
		if err != nil {
			return errors.New(strings.ToTitle("Error unmacing"))
		}
		ok = userlib.HMACEqual(mac_of_data, mainFile[:64])
		if !ok {
			return errors.New(strings.ToTitle("The MAC doesn't match for the main file"))
		}

		decrypted_data := userlib.SymDec(decryptkey, mainFile[64:])

		err = json.Unmarshal(decrypted_data, &filePlacer)
		if err != nil {
			return errors.New(strings.ToTitle("Error Unmarshling (line 1108)"))
		}
		// fetch sharedwith map of file
		sharedMap := filePlacer.SharedWith

		// *******************************************************************************
		// *******************************************************************************

		// . Q: should we regenerate new key for second degree file when we revoke someone's access or
		//      do we continue using old key??????

		// . A: If we do generate new key remember to start storing all keys and salts for second degree file
		// 		including for owner of second degree file in their first degree file!!!!!

		correctKey := filedata.KeyOfSecond
		correctSalt := filedata.SaltOfSecond
		correctSign := userdata.SignKey

		// *******************************************************************************
		// *******************************************************************************

		// lookup recipientUsername in owners's list
		HelperDeleter(sharedMap, recipientUsername, userdata.Username)
		filePlacer.SharedWith = sharedMap
		// finally you are left with map of all the people who still have the file
		// create new random uuid
		contentBytes, err := json.Marshal(filePlacer)
		if err != nil {
			panic(errors.New("cant marshal (line 1135) " + err.Error()))
		}
		randomUUID := uuid.New()
		re_encrypted_contents := userlib.SymEnc(decryptkey, decryptSalt, contentBytes)
		mac_for_file, err := userlib.HMACEval(macKey, re_encrypted_contents)
		if err != nil {
			return errors.New(strings.ToTitle("Error creating mac"))
		}
		maced_second_file := append(mac_for_file[:], re_encrypted_contents[:]...)
		userlib.DatastoreSet(randomUUID, maced_second_file)

		// delete old uuid
		// mac it again
		// overwrite 1st abstraction for self w new random uuid
		filedata.PlaceHolder = randomUUID
		userlib.DatastoreDelete(addressOfSecondAbstraction)
		//create new invite
		HelperNewInviter(sharedMap, userdata.Username, randomUUID, correctKey, correctSalt, correctSign)
		// close
		contentBytes, err = json.Marshal(filedata)
		if err != nil {
			return errors.New(strings.ToTitle("Error marshling (line 1053)"))
		}
		encrypted_contents := userlib.SymEnc(userlib.Argon2Key([]byte(userdata.Password+filename+userdata.Username), userdata.Salt, 16), userdata.Salt, contentBytes)
		signature, err := userlib.DSSign(userdata.SignKey, encrypted_contents)
		if err != nil {
			return errors.New(strings.ToTitle("1098"))
		}
		store := append(signature[:], encrypted_contents[:]...)
		// Store the fake file
		userlib.DatastoreSet(key, store)
	} else {
		return errors.New("An error occurred in Revocation ") //fill in later
	}

	// then lookup recipientUsername as key and remove their whole list and recursively remove all indirect shares

	// keep on deterministically generating their invites for example if bob shared it with eve and mallory, respective invite uuid's would be 'bob-eve-invite' and 'bob-mallory-invite'
	// go to each of these uuids and replace the invite with a new invite that has the new random uuid
	//
	// put filre c
	return nil
}

func NewNode(value []byte, next *Node) *Node {
	var n Node
	n.Val = value
	n.Next = next
	return &n
}

//	func TraverseLinkedList(head *Node) {
//		temp := head
//		for temp != nil {
//			fmt.Printf("%d ", temp.value)
//			temp = temp.next
//		}
//		fmt.Println()
//	}
//
// 1 -> 2 -> 3 -> 4 -> 5
func AddNodeAtEnd(head *Node, data []byte) *Node {
	if head == nil {
		head = NewNode(data, nil)
		return head
	}
	temp := head
	for temp.Next != nil {
		temp = temp.Next
	}
	temp.Next = NewNode(data, nil)
	return head
}

// map[alice:[bob,mallory,charles,eve] bob:[charles] mallory:[] charles:[eve], eve:[]]
func HelperDeleter(dict map[string][]string, recipientUsername string, caller string) {
	userlib.DebugMsg("initial dict: %v", dict)
	if len(dict[recipientUsername]) == 0 {
		return
	}
	for _, child := range dict[recipientUsername] {
		HelperDeleter(dict, child, caller)
		userlib.DebugMsg("dict: %v", dict)
		dict[caller] = remove(dict[caller], child)

	}
	// delete recipientUsername from owners key value list
	dict[caller] = remove(dict[caller], recipientUsername)
	userlib.DebugMsg("dict: %v", dict)

}

func HelperNewInviter(tree map[string][]string, parent string, newKey uuid.UUID, correctKey []byte, correctSalt []byte, correctSign userlib.DSSignKey) {
	for _, child := range tree[parent] {
		oldLocation := parent + "-" + child
		locationInBytes, err := uuid.FromBytes(userlib.Hash([]byte(oldLocation))[:16])
		if err != nil {
			panic(errors.New("cant convert location to bytes " + err.Error()))
		}
		userlib.DatastoreDelete(locationInBytes)
		childPublicKey, ok := userlib.KeystoreGet(child)
		if !ok {
			panic(errors.New("cant find public key of child "))
		}
		var newInvite Invite
		newInvite.LocationOfFile = newKey
		newInvite.NewSourceFile = true
		newInvite.KeyOfFile = correctKey
		newInvite.SaltOfFile = correctSalt

		randomUUIDInvitehash := userlib.Hash([]byte("-" + parent + "-" + child)) // use double --
		randomUUIDInvite, err := uuid.FromBytes(randomUUIDInvitehash[:16])
		if err != nil {
			panic(errors.New(strings.ToTitle("Error creating uuid ")))
		}
		// *** NOW LOAD FILE SHOULD HANDLE THE FETCHING->CHECKING MAC->DECRYPTING->UNMARSHALING
		// create uuid to put invite struct at

		// marshal invite struct
		jsonInvite, err := json.Marshal(newInvite)
		if err != nil {
			panic(errors.New(strings.ToTitle("Error marshling line 984")))
		}
		// encrypt json of invite struct w recipient symm key

		encKeyInviteAdjusted := userlib.Argon2Key([]byte(child+parent), correctSalt, 16)

		saltInviteAdjusted := correctSalt
		// symm enc w random gen key
		encrypted_invite := userlib.SymEnc(encKeyInviteAdjusted, saltInviteAdjusted, jsonInvite)

		signature, err := userlib.DSSign(correctSign, encrypted_invite)
		if err != nil {
			panic(errors.New(strings.ToTitle("Error creating sign 1272")))
		}

		// ATTATCH THE SIGNATURE TO THE BEGINNING OF THE ENCRYPTED STRUCT
		store := append(signature[:], encrypted_invite[:]...)
		// put onto data store
		userlib.DatastoreSet(randomUUIDInvite, store)

		storageKeyForInviteUUID, err := uuid.FromBytes(userlib.Hash([]byte(parent + "-" + child))[:16])
		if err != nil {
			panic(errors.New(strings.ToTitle("Error creating UUID")))
		}
		// first 16 is key next 16 is salt remaining is randomuuid

		tempappend := append(encKeyInviteAdjusted[:], saltInviteAdjusted[:]...)
		information_for_invite := append(tempappend, randomUUIDInvitehash[:16]...)
		information_for_invite_enc, err := userlib.PKEEnc(childPublicKey, information_for_invite)
		if err != nil {
			panic(errors.New(strings.ToTitle("Error Encrypting 997")))
		}
		signatureSec, err := userlib.DSSign(correctSign, information_for_invite_enc)
		if err != nil {
			panic(errors.New(strings.ToTitle("Error creating sign 1294")))
		}

		// ATTATCH THE SIGNATURE TO THE BEGINNING OF THE ENCRYPTED STRUCT
		storeSec := append(signatureSec[:], information_for_invite_enc[:]...)
		// put onto data store
		userlib.DatastoreSet(storageKeyForInviteUUID, storeSec)

		//HelperNewInviter(tree, child, newKey, correctKey, correctSalt, correctSign)
	}
}

func remove(l []string, item string) (b []string) {
	for i, other := range l {
		if other == item {
			b = append(l[:i], l[i+1:]...)
		}
	}
	return b
}

func (userdata *User) helperGetToInvite(sender string) (InviteActual Invite) {
	// Find where the first Invite Abstraction is being kept
	storageKey, err := uuid.FromBytes(userlib.Hash([]byte(sender + "-" + userdata.Username))[:16])
	if err != nil {
		panic(errors.New("Error getting the string+key for the invite in the InviteHelper " + err.Error()))
	}
	// Get the UUID + Key needed to decrypt the real Invite
	dataJSON, ok := userlib.DatastoreGet(storageKey)
	if !ok {
		panic(errors.New(strings.ToTitle("No invite found")))
	}

	// CHECK THE INTREGITY OF THE PLACEHOLDER USING SIGNATURE VERIFICATION
	senderSig, ok := userlib.KeystoreGet("USER" + sender)
	if !ok {
		panic(errors.New(strings.ToTitle("No user public key found for the sender")))
	}

	userSig, ok := userlib.KeystoreGet("USER" + userdata.Username)
	if !ok {
		panic(errors.New(strings.ToTitle("No user public key found for the user")))
	}
	verfied_error_for_sender := userlib.DSVerify(senderSig, dataJSON[256:], dataJSON[0:256])
	verfied_error_for_me := userlib.DSVerify(userSig, dataJSON[256:], dataJSON[0:256])
	if verfied_error_for_sender != nil && verfied_error_for_me != nil {
		panic(errors.New("Error verifying invite in the InviteHelper 1340" + err.Error()))
	}

	// DECRYPT HERE USING THE PUBLIC KEY -------------- THIS NEEDS TO BE DONE ----------------------------------
	inviteInfo, err := userlib.PKEDec(userdata.PrivateKey, dataJSON[256:])
	if err != nil {
		panic(errors.New(strings.ToTitle("Error PKEDec")))
	}
	var keyofInvite []byte
	//var saltOfInvite []byte
	var uuidOfInvite []byte

	keyofInvite = inviteInfo[:16]
	//saltOfInvite = inviteInfo[16:32]
	uuidOfInvite = inviteInfo[32:]

	uuidToFetch, err := uuid.FromBytes(uuidOfInvite[:16])
	if err != nil {
		panic(errors.New(strings.ToTitle("Error creating uuid ")))
	}
	invitedataEnc, ok := userlib.DatastoreGet(uuidToFetch)
	if !ok {
		panic(errors.New(strings.ToTitle("No invite found")))
	}

	verfied_error_for_sender2 := userlib.DSVerify(senderSig, invitedataEnc[256:], invitedataEnc[0:256])
	verfied_error_for_me2 := userlib.DSVerify(userSig, invitedataEnc[256:], invitedataEnc[0:256])
	if verfied_error_for_sender2 != nil && verfied_error_for_me2 != nil {
		panic(errors.New(strings.ToTitle("not able to verify the signature 1368")))
	}
	inviteContent2 := userlib.SymDec(keyofInvite, invitedataEnc[256:])
	if err != nil {
		panic(errors.New("Error getting the string+key for the invite in the InviteHelper " + err.Error()))
	}
	//unmarshal
	var invitePlacer Invite
	err = json.Unmarshal(inviteContent2, &invitePlacer)
	if err != nil {
		panic(errors.New(strings.ToTitle("Error Unmarshling line 1029")))
	}

	return invitePlacer
}

func (userdata *User) updateInvitation(sender string) {
	// Find where the first Invite Abstraction is being kept
	storageKey, err := uuid.FromBytes(userlib.Hash([]byte(sender + "-" + userdata.Username))[:16])
	if err != nil {
		panic(errors.New("Problem getting the UUID of placeholder Invite in the InviteHelper " + err.Error()))
	}
	// Get the UUID + Key needed to decrypt the real Invite
	dataJSON, ok := userlib.DatastoreGet(storageKey)
	if !ok {
		panic(errors.New("Error getting the string+key for the invite in the InviteHelper " + err.Error()))
	}

	// CHECK THE INTREGITY OF THE PLACEHOLDER USING SIGNATURE VERIFICATION
	senderSig, ok := userlib.KeystoreGet("USER" + sender)
	if !ok {
		panic(errors.New("Error getting the string+key for the invite in the InviteHelper " + err.Error()))
	}

	userSig, ok := userlib.KeystoreGet("USER" + userdata.Username)
	if !ok {
		panic(errors.New("Error getting the string+key for the invite in the InviteHelper " + err.Error()))
	}
	verfied_error_for_sender := userlib.DSVerify(senderSig, dataJSON[256:], dataJSON[0:256])
	verfied_error_for_me := userlib.DSVerify(userSig, dataJSON[256:], dataJSON[0:256])
	if verfied_error_for_sender != nil && verfied_error_for_me != nil {
		panic(errors.New("Error verifying invite in the InviteHelper 1409" + err.Error()))
	}

	// DECRYPT HERE USING THE PUBLIC KEY -------------- THIS NEEDS TO BE DONE ----------------------------------
	inviteInfo, err := userlib.PKEDec(userdata.PrivateKey, dataJSON[256:])
	if err != nil {
		panic(errors.New(strings.ToTitle("Error PKEDec")))
	}
	var keyofInvite []byte
	var saltOfInvite []byte
	var uuidOfInvite []byte

	keyofInvite = inviteInfo[:16]
	saltOfInvite = inviteInfo[16:32]
	uuidOfInvite = inviteInfo[32:]

	uuidToFetch, err := uuid.FromBytes(uuidOfInvite[:16])
	if err != nil {
		panic(errors.New(strings.ToTitle("Error creating uuid ")))
	}
	invitedataEnc, ok := userlib.DatastoreGet(uuidToFetch)
	if !ok {
		panic(errors.New(strings.ToTitle("No invite found")))
	}

	verfied_error_for_sender2 := userlib.DSVerify(senderSig, invitedataEnc[256:], invitedataEnc[0:256])
	verfied_error_for_me2 := userlib.DSVerify(userSig, invitedataEnc[256:], invitedataEnc[0:256])
	if verfied_error_for_sender2 != nil && verfied_error_for_me2 != nil {
		panic(errors.New("Error verifying invite in the InviteHelper 1437" + err.Error()))
	}
	inviteContent2 := userlib.SymDec(keyofInvite, invitedataEnc[256:])
	if err != nil {
		panic(errors.New("Error getting the string+key for the invite in the InviteHelper " + err.Error()))
	}
	//unmarshal
	var invitePlacer Invite
	err = json.Unmarshal(inviteContent2, &invitePlacer)
	if err != nil {
		panic(errors.New(strings.ToTitle("Error Unmarshling line 1029")))
	}

	invitePlacer.NewSourceFile = false
	//marshal
	marshaledInvite, err := json.Marshal(invitePlacer)
	if err != nil {
		panic(errors.New("Error Marshling at 1598 " + err.Error()))
	}

	re_encrypting_invite := userlib.SymEnc(keyofInvite, saltOfInvite, marshaledInvite)

	signature, err := userlib.DSSign(userdata.SignKey, re_encrypting_invite)

	if err != nil {
		panic(errors.New(strings.ToTitle("Error Signing the Invite in InviteHelper 1462")))
	}

	store := append(signature[:], re_encrypting_invite[:]...)
	userlib.DatastoreSet(uuidToFetch, store)
}
