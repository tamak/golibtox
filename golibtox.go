package golibtox

// WIP - organ 2014

/*
#cgo LDFLAGS: -ltoxcore

#include <tox/tox.h>
#include <stdlib.h>

// Convenient macro:
// Creates the C function to directly register a given callback
#define HOOK(x) \
static void set_##x(Tox *tox, void *t) { \
	tox_##x(tox, hook_##x, t); \
}

void hook_callback_friend_request(Tox*, uint8_t*, uint8_t*, uint16_t, void*);
void hook_callback_friend_message(Tox*, int32_t, uint8_t*, uint16_t, void*);
void hook_callback_friend_action(Tox*, int32_t, uint8_t*, uint16_t, void*);
void hook_callback_name_change(Tox*, int32_t, uint8_t*, uint16_t, void*);
void hook_callback_status_message(Tox*, int32_t, uint8_t*, uint16_t, void*);
void hook_callback_user_status(Tox*, int32_t, uint8_t, void*);
void hook_callback_typing_change(Tox*, int32_t, uint8_t, void*);
void hook_callback_read_receipt(Tox*, int32_t, uint32_t, void*);
void hook_callback_connection_status(Tox*, int32_t, uint8_t, void*);
void hook_callback_file_send_request(Tox*, int32_t, uint8_t, uint64_t, uint8_t*, uint16_t, void*);
void hook_callback_file_control(Tox*, int32_t, uint8_t, uint8_t, uint8_t, uint8_t*, uint16_t, void*);
void hook_callback_file_data(Tox*, int32_t, uint8_t, uint8_t*, uint16_t, void*);

HOOK(callback_friend_request)
HOOK(callback_friend_message)
HOOK(callback_friend_action)
HOOK(callback_name_change)
HOOK(callback_status_message)
HOOK(callback_user_status)
HOOK(callback_typing_change)
HOOK(callback_read_receipt)
HOOK(callback_connection_status)
HOOK(callback_file_send_request)
HOOK(callback_file_control)
HOOK(callback_file_data)

*/
import "C"

import (
	"encoding/hex"
	"errors"
	"sync"
	"time"
	"unsafe"
)

type FriendRequestFunc func(publicKey []byte, data []byte, length uint16)
type FriendMessageFunc func(friendNumber int32, message []byte, length uint16)
type FriendActionFunc func(friendNumber int32, action []byte, length uint16)
type NameChangeFunc func(friendNumber int32, newName []byte, length uint16)
type StatusMessageFunc func(friendNumber int32, newStatus []byte, length uint16)
type UserStatusFunc func(friendNumber int32, status UserStatus)
type TypingChangeFunc func(friendNumber int32, isTyping bool)
type ReadReceiptFunc func(friendNumber int32, receipt uint32)
type ConnectionStatusFunc func(friendNumber int32, status bool)
type FileSendRequestFunc func(friendNumber int32, filenumber uint8, filesize uint64, filename []byte, filenameLength uint16)
type FileControlFunc func(friendNumber int32, sending bool, filenumber uint8, fileControl FileControl, data []byte, length uint16)
type FileDataFunc func(friendNumber int32, filenumber uint8, data []byte, length uint16)

type Tox struct {
	tox *C.struct_Tox
	mtx sync.Mutex
	// Callbacks
	friendRequestFunc    FriendRequestFunc
	friendMessageFunc    FriendMessageFunc
	friendActionFunc     FriendActionFunc
	nameChangeFunc       NameChangeFunc
	statusMessageFunc    StatusMessageFunc
	userStatusFunc       UserStatusFunc
	typingChangeFunc     TypingChangeFunc
	readReceiptFunc      ReadReceiptFunc
	connectionStatusFunc ConnectionStatusFunc
	fileSendRequestFunc  FileSendRequestFunc
	fileControlFunc      FileControlFunc
	fileDataFunc         FileDataFunc
}

func New() (*Tox, error) {
	ctox := C.tox_new(ENABLE_IPV6_DEFAULT)
	if ctox == nil {
		return nil, errors.New("Error initializing Tox")
	}

	t := &Tox{tox: ctox}

	return t, nil
}

func (t *Tox) Kill() {
	C.tox_kill(t.tox)
}

func (t *Tox) Do() error {
	if t.tox == nil {
		return errors.New("Tox not initialized")
	}

	t.mtx.Lock()
	C.tox_do(t.tox)
	t.mtx.Unlock()

	return nil
}

func (t *Tox) BootstrapFromAddress(address string, port uint16, hexPublicKey string) error {
	if t.tox == nil {
		return errors.New("Tox not initialized")
	}

	caddr := C.CString(address)
	defer C.free(unsafe.Pointer(caddr))

	pubkey, err := hex.DecodeString(hexPublicKey)

	if err != nil {
		return err
	}

	C.tox_bootstrap_from_address(t.tox, caddr, ENABLE_IPV6_DEFAULT, C.htons((C.uint16_t)(port)), (*C.uint8_t)(&pubkey[0]))

	return nil
}

func (t *Tox) IsConnected() (bool, error) {
	if t.tox == nil {
		return false, errors.New("Error getting address, tox not initialized")
	}

	return (C.tox_isconnected(t.tox) == 1), nil
}

func (t *Tox) GetAddress() ([]byte, error) {
	if t.tox == nil {
		return nil, errors.New("Error getting address, tox not initialized")
	}

	address := make([]byte, FRIEND_ADDRESS_SIZE)
	C.tox_get_address(t.tox, (*C.uint8_t)(&address[0]))

	return address, nil
}

func (t *Tox) AddFriend(address []byte, data []byte) (FriendAddError, error) {
	if t.tox == nil {
		return FAERR_UNKNOWN, errors.New("Tox not initialized")
	}

	if len(address) != FRIEND_ADDRESS_SIZE {
		return FAERR_UNKNOWN, errors.New("Error adding friend, wrong size for address")
	}

	faerr := C.tox_add_friend(t.tox, (*C.uint8_t)(&address[0]), (*C.uint8_t)(&data[0]), (C.uint16_t)(len(data)))

	if faerr != 0 {
		return FriendAddError(faerr), errors.New("Error adding friend")
	}

	return FriendAddError(faerr), nil
}

func (t *Tox) AddFriendNorequest(clientId []byte) (int32, error) {
	if t.tox == nil {
		return -1, errors.New("Tox not initialized")
	}

	if len(clientId) != CLIENT_ID_SIZE {
		return -1, errors.New("Incorrect client id")
	}

	n := C.tox_add_friend_norequest(t.tox, (*C.uint8_t)(&clientId[0]))
	if n == -1 {
		return -1, errors.New("Error adding friend")
	}
	return int32(n), nil
}

func (t *Tox) GetFriendNumber(clientId []byte) (int32, error) {
	if t.tox == nil {
		return -1, errors.New("Tox not initialized")
	}
	n := C.tox_get_friend_number(t.tox, (*C.uint8_t)(&clientId[0]))

	return int32(n), nil
}

func (t *Tox) GetClientId(friendNumber int32) ([]byte, error) {
	if t.tox == nil {
		return nil, errors.New("Tox not initialized")
	}
	clientId := make([]byte, CLIENT_ID_SIZE)
	ret := C.tox_get_client_id(t.tox, (C.int32_t)(friendNumber), (*C.uint8_t)(&clientId[0]))

	if ret != 0 {
		return nil, errors.New("Error retrieving client id")
	}

	return clientId, nil
}

func (t *Tox) DelFriend(friendNumber int32) error {
	if t.tox == nil {
		return errors.New("Tox not initialized")
	}
	ret := C.tox_del_friend(t.tox, (C.int32_t)(friendNumber))

	if ret != 0 {
		return errors.New("Error deleting friend")
	}
	return nil
}

func (t *Tox) GetFriendConnectionStatus(friendNumber int32) (bool, error) {
	if t.tox == nil {
		return false, errors.New("Tox not initialized")
	}
	ret := C.tox_get_friend_connection_status(t.tox, (C.int32_t)(friendNumber))
	if ret == -1 {
		return false, errors.New("Error retrieving friend connection status")
	}
	return (int(ret) == 1), nil
}

func (t *Tox) FriendExists(friendNumber int32) (bool, error) {
	if t.tox == nil {
		return false, errors.New("Tox not initialized")
	}
	//int tox_friend_exists(Tox *tox, int32_t friendnumber);
	ret := C.tox_friend_exists(t.tox, (C.int32_t)(friendNumber))

	return (int(ret) == 1), nil
}

func (t *Tox) SendMessage(friendNumber int32, message []byte) (uint32, error) {
	if t.tox == nil {
		return 0, errors.New("Tox not initialized")
	}

	n := C.tox_send_message(t.tox, (C.int32_t)(friendNumber), (*C.uint8_t)(&message[0]), (C.uint32_t)(len(message)))
	if n == 0 {
		return 0, errors.New("Error sending message")
	}
	return uint32(n), nil
}

func (t *Tox) SendMessageWithId(friendNumber int32, id uint32, message []byte) (uint32, error) {
	if t.tox == nil {
		return 0, errors.New("Tox not initialized")
	}

	n := C.tox_send_message_withid(t.tox, (C.int32_t)(friendNumber), (C.uint32_t)(id), (*C.uint8_t)(&message[0]), (C.uint32_t)(len(message)))
	if n == 0 {
		return 0, errors.New("Error sending message")
	}
	return uint32(n), nil
}

func (t *Tox) SendAction(friendNumber int32, action []byte) (uint32, error) {
	if t.tox == nil {
		return 0, errors.New("Tox not initialized")
	}

	n := C.tox_send_action(t.tox, (C.int32_t)(friendNumber), (*C.uint8_t)(&action[0]), (C.uint32_t)(len(action)))
	if n == 0 {
		return 0, errors.New("Error sending action")
	}
	return uint32(n), nil
}

func (t *Tox) SendActionWithId(friendNumber int32, id uint32, action []byte) (uint32, error) {
	if t.tox == nil {
		return 0, errors.New("Tox not initialized")
	}

	n := C.tox_send_message_withid(t.tox, (C.int32_t)(friendNumber), (C.uint32_t)(id), (*C.uint8_t)(&action[0]), (C.uint32_t)(len(action)))
	if n == 0 {
		return 0, errors.New("Error sending action")
	}
	return uint32(n), nil
}

func (t *Tox) SetName(name string) error {
	if t.tox == nil {
		return errors.New("Tox not initialized")
	}

	ret := C.tox_set_name(t.tox, (*C.uint8_t)(&[]byte(name)[0]), (C.uint16_t)(len(name)))
	if ret != 0 {
		return errors.New("Error setting name")
	}
	return nil
}

func (t *Tox) GetSelfName() (string, error) {
	if t.tox == nil {
		return "", errors.New("Tox not initialized")
	}

	cname := make([]byte, MAX_NAME_LENGTH)

	n := C.tox_get_self_name(t.tox, (*C.uint8_t)(&cname[0]))
	if n == 0 {
		return "", errors.New("Error retrieving self name")
	}

	name := string(cname[:n])

	return name, nil
}

func (t *Tox) GetName(friendNumber int32) (string, error) {
	if t.tox == nil {
		return "", errors.New("Tox not initialized")
	}

	cname := make([]byte, MAX_NAME_LENGTH)

	n := C.tox_get_name(t.tox, (C.int32_t)(friendNumber), (*C.uint8_t)(&cname[0]))
	if n == -1 {
		return "", errors.New("Error retrieving name")
	}

	name := string(cname[:n])

	return name, nil
}

func (t *Tox) GetNameSize(friendNumber int32) (int, error) {
	if t.tox == nil {
		return -1, errors.New("tox not initialized")
	}

	ret := C.tox_get_name_size(t.tox, (C.int32_t)(friendNumber))
	if ret == -1 {
		return -1, errors.New("Error retrieving name size")
	}

	return int(ret), nil
}

func (t *Tox) GetSelfNameSize() (int, error) {
	if t.tox == nil {
		return -1, errors.New("tox not initialized")
	}

	ret := C.tox_get_self_name_size(t.tox)
	if ret == -1 {
		return -1, errors.New("Error retrieving self name size")
	}

	return int(ret), nil
}

func (t *Tox) SetStatusMessage(status []byte) error {
	if t.tox == nil {
		return errors.New("Tox not initialized")
	}

	ret := C.tox_set_status_message(t.tox, (*C.uint8_t)(&status[0]), (C.uint16_t)(len(status)))
	if ret != 0 {
		return errors.New("Error setting status message")
	}
	return nil
}

func (t *Tox) SetUserStatus(status UserStatus) error {
	if t.tox == nil {
		return errors.New("Tox not initialized")
	}

	ret := C.tox_set_user_status(t.tox, (C.uint8_t)(status))
	if ret != 0 {
		return errors.New("Error setting status")
	}
	return nil
}

func (t *Tox) GetStatusMessageSize(friendNumber int32) (int, error) {
	if t.tox == nil {
		return -1, errors.New("tox not initialized")
	}

	ret := C.tox_get_status_message_size(t.tox, (C.int32_t)(friendNumber))
	if ret == -1 {
		return -1, errors.New("Error retrieving status message size")
	}

	return int(ret), nil
}

func (t *Tox) GetSelfStatusMessageSize() (int, error) {
	if t.tox == nil {
		return -1, errors.New("tox not initialized")
	}

	ret := C.tox_get_self_status_message_size(t.tox)
	if ret == -1 {
		return -1, errors.New("Error retrieving self status message size")
	}

	return int(ret), nil
}

func (t *Tox) GetStatusMessage(friendNumber int32) ([]byte, error) {
	if t.tox == nil {
		return nil, errors.New("Tox not initialized")
	}

	status := make([]byte, MAX_STATUSMESSAGE_LENGTH)

	n := C.tox_get_status_message(t.tox, (C.int32_t)(friendNumber), (*C.uint8_t)(&status[0]), MAX_STATUSMESSAGE_LENGTH)
	if n == -1 {
		return nil, errors.New("Error retrieving status message")
	}

	// Truncate status to n-byte read
	status = status[:n]

	return status, nil
}

func (t *Tox) GetSelfStatusMessage() ([]byte, error) {
	if t.tox == nil {
		return nil, errors.New("Tox not initialized")
	}

	status := make([]byte, MAX_STATUSMESSAGE_LENGTH)

	n := C.tox_get_self_status_message(t.tox, (*C.uint8_t)(&status[0]), MAX_STATUSMESSAGE_LENGTH)
	if n == -1 {
		return nil, errors.New("Error retrieving self status message")
	}

	// Truncate status to n-byte read
	status = status[:n]

	return status, nil
}

func (t *Tox) GetUserStatus(friendNumber int32) (UserStatus, error) {
	if t.tox == nil {
		return USERSTATUS_INVALID, errors.New("Tox not initialized")
	}
	n := C.tox_get_user_status(t.tox, (C.int32_t)(friendNumber))

	return UserStatus(n), nil
}

func (t *Tox) GetSelfUserStatus() (UserStatus, error) {
	if t.tox == nil {
		return USERSTATUS_INVALID, errors.New("Tox not initialized")
	}
	n := C.tox_get_self_user_status(t.tox)

	return UserStatus(n), nil
}

func (t *Tox) GetLastOnline(friendNumber int32) (time.Time, error) {
	if t.tox == nil {
		return time.Time{}, errors.New("Tox not initialized")
	}
	ret := C.tox_get_last_online(t.tox, (C.int32_t)(friendNumber))

	if int(ret) == -1 {
		return time.Time{}, errors.New("Error getting last online time")
	}

	last := time.Unix(int64(ret), 0)

	return last, nil
}

func (t *Tox) SetUserIsTyping(friendNumber int32, isTyping bool) error {
	if t.tox == nil {
		return errors.New("Tox not initialized")
	}
	typing := 0
	if isTyping {
		typing = 1
	}

	ret := C.tox_set_user_is_typing(t.tox, (C.int32_t)(friendNumber), (C.uint8_t)(typing))

	if ret != 0 {
		return errors.New("Error setting typing status")
	}

	return nil
}

func (t *Tox) GetIsTyping(friendNumber int32) (bool, error) {
	if t.tox == nil {
		return false, errors.New("Tox not initialized")
	}

	ret := C.tox_get_is_typing(t.tox, (C.int32_t)(friendNumber))

	return (ret == 1), nil
}

func (t *Tox) SetSendsReceipts(friendNumber int32, send bool) error {
	if t.tox == nil {
		return errors.New("Tox not initialized")
	}
	csend := 0
	if send {
		csend = 1
	}

	C.tox_set_sends_receipts(t.tox, (C.int32_t)(friendNumber), (C.int)(csend))

	return nil
}

func (t *Tox) CountFriendlist() (uint32, error) {
	if t.tox == nil {
		return 0, errors.New("Tox not initialized")
	}
	n := C.tox_count_friendlist(t.tox)

	return uint32(n), nil
}

func (t *Tox) GetNumOnlineFriends() (uint32, error) {
	if t.tox == nil {
		return 0, errors.New("Tox not initialized")
	}
	n := C.tox_get_num_online_friends(t.tox)

	return uint32(n), nil
}

func (t *Tox) GetFriendlist() ([]int32, error) {
	if t.tox == nil {
		return nil, errors.New("Tox not initialized")
	}

	size, _ := t.CountFriendlist()
	cfriendlist := make([]int32, size)

	n := C.tox_get_friendlist(t.tox, (*C.int32_t)(&cfriendlist[0]), (C.uint32_t)(size))

	friendlist := cfriendlist[:n]

	return friendlist, nil
}

func (t *Tox) GetNospam() (uint32, error) {
	if t.tox == nil {
		return 0, errors.New("Tox not initialized")
	}

	n := C.tox_get_nospam(t.tox)

	return uint32(n), nil
}

func (t *Tox) SetNospam(nospam uint32) error {
	if t.tox == nil {
		return errors.New("Tox not initialized")
	}

	C.tox_set_nospam(t.tox, (C.uint32_t)(nospam))

	return nil
}

func (t *Tox) NewFileSender(friendNumber int32, filesize uint64, filename []byte) (int, error) {
	if t.tox == nil {
		return -1, errors.New("Tox not initialized")
	}

	if len(filename) > 255 {
		return -1, errors.New("Filename too long")
	}

	n := C.tox_new_file_sender(t.tox, (C.int32_t)(friendNumber), (C.uint64_t)(filesize), (*C.uint8_t)(&filename[0]), (C.uint16_t)(len(filename)))

	if n == -1 {
		return -1, errors.New("Error sending file request")
	}

	return int(n), nil
}

func (t *Tox) FileSendControl(friendNumber int32, receiving bool, filenumber uint8, messageId FileControl, data []byte) error {
	if t.tox == nil {
		return errors.New("Tox not initialized")
	}

	cReceiving := 0
	if receiving {
		cReceiving = 1
	}

	// Stupid workaround to prevent index out of range when using &data[0] if data == nil
	var cdata *C.uint8_t
	var clen C.uint16_t

	if data == nil {
		cdata = nil
		clen = 0
	} else {
		cdata = (*C.uint8_t)(&data[0])
		clen = (C.uint16_t)(len(data))
	}
	// End of stupid workaround

	n := C.tox_file_send_control(t.tox, (C.int32_t)(friendNumber), (C.uint8_t)(cReceiving), (C.uint8_t)(filenumber), (C.uint8_t)(messageId), cdata, clen)

	if n == -1 {
		return errors.New("Error sending file control")
	}

	return nil
}

func (t *Tox) FileSendData(friendNumber int32, filenumber uint8, data []byte) error {
	if t.tox == nil {
		return errors.New("Tox not initialized")
	}

	if len(data) == 0 {
		return errors.New("Error sending empty data")

	}

	n := C.tox_file_send_data(t.tox, (C.int32_t)(friendNumber), (C.uint8_t)(filenumber), (*C.uint8_t)(&data[0]), (C.uint16_t)(len(data)))

	if n == -1 {
		return errors.New("Error sending file data, is data too big ?")
	}

	return nil
}

func (t *Tox) FileDataSize(friendNumber int32) (int, error) {
	if t.tox == nil {
		return -1, errors.New("Tox not initialized")
	}

	n := C.tox_file_data_size(t.tox, (C.int32_t)(friendNumber))

	if n == -1 {
		return -1, errors.New("Error getting file data size")
	}

	return int(n), nil
}

func (t *Tox) FileDataRemaining(friendNumber int32, filenumber uint8, receiving bool) (uint64, error) {
	if t.tox == nil {
		return 0, errors.New("Tox not initialized")
	}

	cReceiving := 0
	if receiving {
		cReceiving = 1
	}

	n := C.tox_file_data_remaining(t.tox, (C.int32_t)(friendNumber), (C.uint8_t)(filenumber), (C.uint8_t)(cReceiving))

	if n == 0 {
		return 0, errors.New("Error sending file control")
	}

	return uint64(n), nil
}

func (t *Tox) Size() (uint32, error) {
	if t.tox == nil {
		return 0, errors.New("tox not initialized")
	}

	return uint32(C.tox_size(t.tox)), nil
}

func (t *Tox) Save() ([]byte, error) {
	if t.tox == nil {
		return nil, errors.New("tox not initialized")
	}
	size, _ := t.Size()

	data := make([]byte, size)
	C.tox_save(t.tox, (*C.uint8_t)(&data[0]))

	return data, nil

}

func (t *Tox) Load(data []byte) error {
	if t.tox == nil {
		return errors.New("tox not initialized")
	}

	ret := C.tox_load(t.tox, (*C.uint8_t)(&data[0]), (C.uint32_t)(len(data)))

	if ret == -1 {
		return errors.New("Error loading data")
	}
	return nil
}

func (t *Tox) CallbackFriendRequest(f FriendRequestFunc) {
	if t.tox != nil {
		t.friendRequestFunc = f
		C.set_callback_friend_request(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackFriendMessage(f FriendMessageFunc) {
	if t.tox != nil {
		t.friendMessageFunc = f
		C.set_callback_friend_message(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackFriendAction(f FriendActionFunc) {
	if t.tox != nil {
		t.friendActionFunc = f
		C.set_callback_friend_action(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackNameChange(f NameChangeFunc) {
	if t.tox != nil {
		t.nameChangeFunc = f
		C.set_callback_name_change(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackStatusMessage(f StatusMessageFunc) {
	if t.tox != nil {
		t.statusMessageFunc = f
		C.set_callback_status_message(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackUserStatus(f UserStatusFunc) {
	if t.tox != nil {
		t.userStatusFunc = f
		C.set_callback_user_status(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackTypingChange(f TypingChangeFunc) {
	if t.tox != nil {
		t.typingChangeFunc = f
		C.set_callback_typing_change(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackReadReceipt(f ReadReceiptFunc) {
	if t.tox != nil {
		t.readReceiptFunc = f
		C.set_callback_read_receipt(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackConnectionStatus(f ConnectionStatusFunc) {
	if t.tox != nil {
		t.connectionStatusFunc = f
		C.set_callback_connection_status(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackFileSendRequest(f FileSendRequestFunc) {
	if t.tox != nil {
		t.fileSendRequestFunc = f
		C.set_callback_file_send_request(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackFileControl(f FileControlFunc) {
	if t.tox != nil {
		t.fileControlFunc = f
		C.set_callback_file_control(t.tox, unsafe.Pointer(t))
	}
}

func (t *Tox) CallbackFileData(f FileDataFunc) {
	if t.tox != nil {
		t.fileDataFunc = f
		C.set_callback_file_data(t.tox, unsafe.Pointer(t))
	}
}
