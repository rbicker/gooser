// Code generated by running "go generate" in golang.org/x/text. DO NOT EDIT.

package translations

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

type dictionary struct {
	index []uint32
	data  string
}

func (d *dictionary) Lookup(key string) (data string, ok bool) {
	p := messageKeyToIndex[key]
	start, end := d.index[p], d.index[p+1]
	if start == end {
		return "", false
	}
	return d.data[start:end], true
}

func init() {
	dict := map[string]catalog.Dictionary{
		"de": &dictionary{index: deIndex, data: deData},
		"en": &dictionary{index: enIndex, data: enData},
	}
	fallback := language.MustParse("en")
	cat, err := catalog.NewFromMap(dict, catalog.Fallback(fallback))
	if err != nil {
		panic(err)
	}
	message.DefaultCatalog = cat
}

var messageKeyToIndex = map[string]int{
	"%s has a length of 0":     4,
	"%s: confirm mail address": 34,
	"%s: password reset":       37,
	"Hi %s! Please confirm your mail address by clicking the following link. Thanks!\n%s":                                                                35,
	"Hi %s! To reset your password, click the following link: \n%s\n\nIf you did not request to reset your password, please ignore this message. Thanks": 38,
	"could not find group with id %s":                                  39,
	"could not find user with id %s":                                   49,
	"could not parse given language":                                   51,
	"error while querying %s":                                          10,
	"error while querying member":                                      43,
	"error while saving group":                                         17,
	"error while saving user":                                          31,
	"error while sending mail: %s":                                     36,
	"group name needs to have a length of at least 3":                  40,
	"internal error while building filter":                             0,
	"invalid group id '%s'":                                            16,
	"invalid id '%s'":                                                  13,
	"invalid mail address":                                             52,
	"invalid page token given":                                         5,
	"invalid rsql filter string '%s': %s":                              3,
	"invalid token":                                                    23,
	"invalid user id":                                                  32,
	"invalid user id '%s'":                                             30,
	"invalid username, only lowercase letters and numbers are allowed": 50,
	"mail address not set":                                             55,
	"no token given":                                                   21,
	"not allowed to change password for other users":                   64,
	"not allowed to create groups":                                     44,
	"not allowed to delete groups":                                     48,
	"not allowed to delete user":                                       62,
	"not allowed to edit other users":                                  58,
	"not allowed to set confirmed":                                     56,
	"not allowed to update groups":                                     45,
	"only %v of %v given memberIds were found":                         42,
	"orderBy field has a length of 0":                                  1,
	"pagination filter and given filters do not match":                 6,
	"pagination orderBy and given orderBy do not match":                7,
	"password cannot be changed using the UpdateUser function, use ChangePassword instead": 60,
	"password mismatch":                                              65,
	"password must have a length of at least 7":                      53,
	"roles cannot be assigned to users directly":                     59,
	"roles cannot be assigned to users directly, use groups instead": 54,
	"the request was canceled by the client":                         8,
	"token mismatch":                                                 22,
	"unable to count %s":                                             9,
	"unable to count groups":                                         12,
	"unable to count users":                                          27,
	"unable to create generate field mask: %s":                       46,
	"unable to decode group: %s":                                     11,
	"unable to decode user: %s":                                      26,
	"unable to encrypt confirmation: %s":                             20,
	"unable to encrypt reset password struct: %s":                    25,
	"unable to find group named %s":                                  15,
	"unable to find group with id %s":                                14,
	"unable to find group with id '%s'":                              18,
	"unable to find user":                                            29,
	"unable to find user with given id":                              33,
	"unable to find user with id %s":                                 28,
	"unable to hash given password":                                  57,
	"unable to json marshal confirmation: %s":                        19,
	"unable to json marshal reset password struct: %s":               24,
	"unable to merge groups":                                         47,
	"unable to merge users":                                          61,
	"unable to query members":                                        41,
	"unable to remove user from group %s":                            63,
	"unable to save user":                                            66,
	"unable to search next document while creating pagination token": 2,
	"unable to send reset password mail":                             67,
}

var deIndex = []uint32{ // 69 elements
	// Entry 0 - 1F
	0x00000000, 0x0000002b, 0x0000004d, 0x000000aa,
	0x000000d8, 0x000000f4, 0x0000011a, 0x00000158,
	0x0000019d, 0x000001c6, 0x000001e4, 0x00000203,
	0x0000022f, 0x00000255, 0x00000269, 0x0000029a,
	0x000002cb, 0x000002e9, 0x0000030a, 0x0000033d,
	0x00000371, 0x000003a8, 0x000003bd, 0x000003d9,
	0x000003eb, 0x00000428, 0x00000468, 0x00000497,
	0x000004be, 0x000004f3, 0x00000519, 0x00000538,
	// Entry 20 - 3F
	0x0000055c, 0x00000573, 0x000005ae, 0x000005ce,
	0x00000637, 0x0000065e, 0x0000067c, 0x00000736,
	0x00000769, 0x000007a7, 0x000007d1, 0x00000800,
	0x00000823, 0x0000084a, 0x00000875, 0x000008a3,
	0x000008d1, 0x000008f7, 0x0000092a, 0x00000970,
	0x00000995, 0x000009ad, 0x000009e6, 0x00000a35,
	0x00000a50, 0x00000a75, 0x00000aab, 0x00000adf,
	0x00000b17, 0x00000b8d, 0x00000bbc, 0x00000be7,
	// Entry 40 - 5F
	0x00000c16, 0x00000c57, 0x00000c76, 0x00000c9d,
	0x00000cce,
} // Size: 300 bytes

const deData string = "" + // Size: 3278 bytes
	"\x02Interner Fehler beim Erstellen des Filters\x02Sortierfeld hat eine L" +
	"änge von 0\x02während dem Erstellen des Pagination-Tokens konnte das Fo" +
	"lgedokument nicht abgefragt werden\x02ungültiger rsql Filter String '%[1" +
	"]s': %[2]s\x02%[1]s hat eine Länge von 0\x02Ungültiger Pagination Token " +
	"erhalten\x02Pagination Filter und gegebener Filter stimmen nicht überein" +
	"\x02Pagination Sortierung und gegebene Sortierung stimmen nicht überein" +
	"\x02die Anfrage wurde vom Client abgebrochen\x02Fehler beim Zählen von %" +
	"[1]s\x02Fehler beim Abfragen von %[1]s\x02Gruppe konnte nicht decodiert " +
	"werden: %[1]s\x02Gruppen konnten nicht gezählt werden\x02ungültige ID %[" +
	"1]s\x02Gruppe mit id %[1]s konnte nicht gefunden werden\x02Gruppe namens" +
	" %[1]s konnte nicht gefunden werden\x02Ungültige Gruppen-ID '%[1]s'\x02F" +
	"ehler beim Speichern der Gruppe\x02Gruppe mit ID '%[1]s' konnte nicht ge" +
	"funden werden\x02Bestätigung konnte nicht umgewandelt werden: %[1]s\x02B" +
	"estätigung konnte nicht verschlüsselt werden: %[1]s\x02Kein Token angege" +
	"ben\x02Token stimmt nicht überein\x02ungültiger Token\x02Passwort Reset " +
	"Objekt konnte nicht umgewandelt werden: %[1]s\x02Passwort Reset Objekt k" +
	"onnte nicht verschlüsselt werden: %[1]s\x02Benutzer konnten nicht dekodi" +
	"ert werden: %[1]s\x02Benutzer konnten nicht gezählt werden\x02Benutzer m" +
	"it ID '%[1]s' konnte nicht gefunden werden\x02Benutzer konnte nicht gefu" +
	"nden werden\x02Ungültige Benutzer ID '%[1]s'\x02Fehler beim Speichern de" +
	"s Benutzers\x02Ungültige Benutzer ID\x02Benutzer mit der gegebenen ID ko" +
	"nnte nicht gefunden werden\x02%[1]s: Mail-Adresse bestätigen\x02Hallo %[" +
	"1]s! Bitte bestätige deine Mail-Adresse, indem du auf den folgenden Link" +
	" klickst. Danke!\x0a%[2]s \x02Fehler beim Versenden des Mails: %[1]s\x02" +
	"%[1]s: Passwort zurücksetzen\x02Hallo %[1]s! Um dein Passwort zurückzuse" +
	"tzen, klicke den folgenden Link: \x0a%[2]s\x0a\x0aFalls du das zurückset" +
	"zen des Passworts nicht angefordert hast, bitte ignoriere diese Nachrich" +
	"t. Danke\x02Gruppe mit ID '%[1]s' konnte nicht gefunden werden\x02Name d" +
	"er Gruppe sollte mindestens eine Länge von 3 aufweisen\x02Mitglieder kon" +
	"nten nicht abgefragt werden\x02Nur %[1]v der %[2]v Mitglieder wurden gef" +
	"unden\x02Fehler beim Abfragen des Mitglieds\x02Nicht berechtigt, Gruppen" +
	" zu erstellen\x02Nicht berechtigt, Gruppen zu aktualisieren\x02Feldmaske" +
	" konnte nicht erstellt werden: %[1]s\x02Gruppen konnten nicht zusammenge" +
	"führt werden\x02Nicht berechtigt, Gruppen zu löschen\x02Benutzer mit id " +
	"%[1]s konnte nicht gefunden werden\x02Ungüliger Benutzername, nur Kleinb" +
	"uchstaben und Nummern sind erlaubt\x02Sprache konnte nicht bestimmt werd" +
	"en\x02Ungültige Mail Adresse\x02Das Passwort muss mindestens eine Länge " +
	"von 7 aufweisen\x02Rollen können nicht direkt Benutzern zugewiesen werde" +
	"n, verwende Gruppen dazu\x02Mail Adresse nicht gegeben\x02Bestätigt darf" +
	" nicht gesetzt werden\x02Es konnte kein Hash für das Passwort erstellt w" +
	"erden\x02Keine Berechtigung um andere Benutzer zu bearbeiten\x02Rollen k" +
	"önnen nicht direkt Benutzern zugeordnet werden\x02Passwort kann nicht m" +
	"it der UpdateUser Funktion aktualisiert werden, verwende die ChangePassw" +
	"ord Funktion stattdessen\x02Benutzer können nicht zusammengeführt werden" +
	"\x02Keine Berechtigung um Benutzer zu löschen\x02Benutzer kann nicht von" +
	" Gruppe entfernt werden\x02Keine Berechtigungen um das Passwort anderer " +
	"Benutzer zu ändern\x02Passwort stimmt nicht überein\x02Benutzer kann nic" +
	"ht gespeichert werden\x02Passwort Reset Mail konnte nicht versandt werde" +
	"n"

var enIndex = []uint32{ // 69 elements
	// Entry 0 - 1F
	0x00000000, 0x00000025, 0x00000045, 0x00000084,
	0x000000ae, 0x000000c6, 0x000000df, 0x00000110,
	0x00000142, 0x00000169, 0x0000017f, 0x0000019a,
	0x000001b8, 0x000001cf, 0x000001e2, 0x00000205,
	0x00000226, 0x0000023f, 0x00000258, 0x0000027d,
	0x000002a8, 0x000002ce, 0x000002dd, 0x000002ec,
	0x000002fa, 0x0000032e, 0x0000035d, 0x0000037a,
	0x00000390, 0x000003b2, 0x000003c6, 0x000003de,
	// Entry 20 - 3F
	0x000003f6, 0x00000406, 0x00000428, 0x00000444,
	0x0000049d, 0x000004bd, 0x000004d3, 0x00000569,
	0x0000058c, 0x000005bc, 0x000005d4, 0x00000603,
	0x0000061f, 0x0000063c, 0x00000659, 0x00000685,
	0x0000069c, 0x000006b9, 0x000006db, 0x0000071c,
	0x0000073b, 0x00000750, 0x0000077a, 0x000007b9,
	0x000007ce, 0x000007eb, 0x00000809, 0x00000829,
	0x00000854, 0x000008a9, 0x000008bf, 0x000008da,
	// Entry 40 - 5F
	0x00000901, 0x00000930, 0x00000942, 0x00000956,
	0x00000979,
} // Size: 300 bytes

const enData string = "" + // Size: 2425 bytes
	"\x02internal error while building filter\x02orderBy field has a length o" +
	"f 0\x02unable to search next document while creating pagination token" +
	"\x02invalid rsql filter string '%[1]s': %[2]s\x02%[1]s has a length of 0" +
	"\x02invalid page token given\x02pagination filter and given filters do n" +
	"ot match\x02pagination orderBy and given orderBy do not match\x02the req" +
	"uest was canceled by the client\x02unable to count %[1]s\x02error while " +
	"querying %[1]s\x02unable to decode group: %[1]s\x02unable to count group" +
	"s\x02invalid id '%[1]s'\x02unable to find group with id %[1]s\x02unable " +
	"to find group named %[1]s\x02invalid group id '%[1]s'\x02error while sav" +
	"ing group\x02unable to find group with id '%[1]s'\x02unable to json mars" +
	"hal confirmation: %[1]s\x02unable to encrypt confirmation: %[1]s\x02no t" +
	"oken given\x02token mismatch\x02invalid token\x02unable to json marshal " +
	"reset password struct: %[1]s\x02unable to encrypt reset password struct:" +
	" %[1]s\x02unable to decode user: %[1]s\x02unable to count users\x02unabl" +
	"e to find user with id %[1]s\x02unable to find user\x02invalid user id '" +
	"%[1]s'\x02error while saving user\x02invalid user id\x02unable to find u" +
	"ser with given id\x02%[1]s: confirm mail address\x02Hi %[1]s! Please con" +
	"firm your mail address by clicking the following link. Thanks!\x0a%[2]s" +
	"\x02error while sending mail: %[1]s\x02%[1]s: password reset\x02Hi %[1]s" +
	"! To reset your password, click the following link: \x0a%[2]s\x0a\x0aIf " +
	"you did not request to reset your password, please ignore this message. " +
	"Thanks\x02could not find group with id %[1]s\x02group name needs to have" +
	" a length of at least 3\x02unable to query members\x02only %[1]v of %[2]" +
	"v given memberIds were found\x02error while querying member\x02not allow" +
	"ed to create groups\x02not allowed to update groups\x02unable to create " +
	"generate field mask: %[1]s\x02unable to merge groups\x02not allowed to d" +
	"elete groups\x02could not find user with id %[1]s\x02invalid username, o" +
	"nly lowercase letters and numbers are allowed\x02could not parse given l" +
	"anguage\x02invalid mail address\x02password must have a length of at lea" +
	"st 7\x02roles cannot be assigned to users directly, use groups instead" +
	"\x02mail address not set\x02not allowed to set confirmed\x02unable to ha" +
	"sh given password\x02not allowed to edit other users\x02roles cannot be " +
	"assigned to users directly\x02password cannot be changed using the Updat" +
	"eUser function, use ChangePassword instead\x02unable to merge users\x02n" +
	"ot allowed to delete user\x02unable to remove user from group %[1]s\x02n" +
	"ot allowed to change password for other users\x02password mismatch\x02un" +
	"able to save user\x02unable to send reset password mail"

	// Total table size 6303 bytes (6KiB); checksum: 11146F60
