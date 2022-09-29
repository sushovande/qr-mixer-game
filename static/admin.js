/**
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

function userChanged() {
  console.log("change processing");

  /** @type HTMLTextAreaElement */
  const ta = document.getElementById('users');
  const { headers, words } = parseTSV(ta.value)

  if (!validate(headers, words)) {
    return;
  }
  updateTextFromStructs(headers, words);
}

function generateQrCodeButtonClicked() {
  /** @type HTMLTextAreaElement */
  const ta = document.getElementById('users');
  const { headers, words } = parseTSV(ta.value)
  if (!generateQrCodes(headers, words)) {
    return;
  }
  updateTextFromStructs(headers, words);
}

function saveButtonClicked() {
  /** @type HTMLTextAreaElement */
  const ta = document.getElementById('users');
  const { headers, words } = parseTSV(ta.value)
  if (!validate(headers, words)) {
    return;
  }
  if (generateQrCodes(headers, words)) {
    updateTextFromStructs(headers, words);
  }
  const data = generateConsolidatedObject(headers, words);

  const postData = new URLSearchParams({ "users": JSON.stringify(data) });
  fetch(PostEndpoint, { method: 'post', body: postData })
    .then(response => {
      if (!response.ok) {
        response.text().then(p => { document.getElementById('errormsg').textContent = 'Error: ' + p });
      } else {
        response.text().then(p => { document.getElementById('errormsg').textContent = 'OK: ' + p });
      }
    }).catch((error) => {
      document.getElementById('errormsg').textContent = 'Error: ' + error;
    });
}

/**
 * updates the text of the users textarea based on the parsed TSV values.
 * @param {Array<string>} headers - the column headers (first row)
 * @param {Array<Array<string>>} words - the entries in the rest of the table. 
 */
function updateTextFromStructs(headers, words) {
  /** @type HTMLTextAreaElement */
  const ta = document.getElementById('users');
  let trimmedText = '';
  trimmedText += headers.join('\t');
  for (let line of words) {
    trimmedText += '\n' + line.join('\t');
  }

  ta.value = trimmedText;
}

/**
 * Reads in a text block as TSV and produces headers and cell values
 * @param {string} text - the text to parse
 * @returns {Object} with two params: headers, which is a string[], and words which is a string[][]
 */
function parseTSV(text) {
  const lines = text.split('\n');
  /** @type Array<string> */
  const headers = [];
  /** @type Array<Array<string>> */
  const words = [];
  let trimmedText = '';
  for (let i = 0; i < lines.length; i++) {
    lines[i] = lines[i].trim();
    if (lines[i].length > 0) {
      trimmedText += lines[i] + '\n';
      if (i == 0) {
        let frags = lines[i].split('\t');
        for (let j = 0; j < frags.length; j++) {
          headers[j] = frags[j];
        }
      } else {
        let frags = lines[i].split('\t');
        words.push(frags);
      }
    }
  }

  return {
    headers: headers,
    words: words
  }
}

/**
 * Generates the random QR code values and other associated values like card suit and rank.
 * @param {Array<string>} headers - the column headers (first row)
 * @param {Array<Array<string>>} words - the entries in the rest of the table. 
 * @returns {boolean} - if something needed to be generated.
 */
function generateQrCodes(headers, words) {
  if (!validate(headers, words)) {
    return false;
  }

  let dirty = false;

  let qrCodeIndex = -1;
  let cardSuiteIndex = -1;
  let cardRankIndex = -1;
  for (let i = 0; i < headers.length; i++) {
    if (headers[i].toLowerCase() === 'qrcode') {
      qrCodeIndex = i;
    } else if (headers[i].toLowerCase() === 'cardsuit') {
      cardSuiteIndex = i;
    } else if (headers[i].toLowerCase() === 'cardrank') {
      cardRankIndex = i;
    }
  }

  if (qrCodeIndex == -1) {
    headers.push('qrcode');
    dirty = true;
    qrCodeIndex = headers.length - 1;
  }
  for (let i = 0; i < words.length; i++) {
    if (!(qrCodeIndex in words[i]) || words[i][qrCodeIndex] === "") {
      words[i][qrCodeIndex] = SITE_URL_PREFIX + "#" + getRandomString(10);
      dirty = true;
    }
  }


  if (cardSuiteIndex == -1) {
    headers.push('cardsuit');
    dirty = true;
    cardSuiteIndex = headers.length - 1;
  }
  for (let i = 0; i < words.length; i++) {
    if (!(cardSuiteIndex in words[i]) || words[i][cardSuiteIndex] === "") {
      words[i][cardSuiteIndex] = getRandomCardSuit();
      dirty = true;
    }
  }

  if (cardRankIndex == -1) {
    headers.push('cardrank');
    dirty = true;
    cardRankIndex = headers.length - 1;
  }
  for (let i = 0; i < words.length; i++) {
    if (!(cardRankIndex in words[i]) || words[i][cardRankIndex] === "") {
      words[i][cardRankIndex] = (getRandomInt(13) + 1);
      dirty = true;
    }
  }

  // return true if at least something changed
  return dirty;
}

/**
 * @param {number} max - exclusive upper bound.
 * @returns {number} a random int in the range [0, max).
 */
function getRandomInt(max) {
  return Math.floor(Math.random() * max);
}

/**
 * @param {number} length - how long of a string we want.
 * @returns {string} a random string with lowercase alpha and numbers of specified length.
 */
function getRandomString(length) {
  const template = 'abcdefghijklmnopqrstuvwxyz0123456789';
  let sz = '';
  for (let i = 0; i < length; i++) {
    sz += template[getRandomInt(36)];
  }
  return sz;
}

/**
 * @returns {string} a random choice among (SPADES, HEARTS, CLUBS, DIAMONDS).
 */
function getRandomCardSuit() {
  const choices = ['SPADES', 'HEARTS', 'CLUBS', 'DIAMONDS'];
  return choices[getRandomInt(4)];
}

/**
 * validate checks if the pasted in spreadsheet satisfies some basic checks.
 * @param {Array<string>} headers - the column headers (first row)
 * @param {Array<Array<string>>} words - the entries in the rest of the table. 
 * @returns {boolean} true if successful
 */
function validate(headers, words) {
  const vt = document.getElementById('validation');
  vt.textContent = 'OK';
  if (headers.length < 2) {
    vt.textContent = 'ERROR: At least two columns (Name, Username) are required.';
    return false;
  }

  let nameIndex = -1;
  let usernameIndex = -1;
  for (let i = 0; i < headers.length; i++) {
    if (headers[i].toLowerCase() === 'name') {
      nameIndex = i;
    } else if (headers[i].toLowerCase() === 'username') {
      usernameIndex = i;
    }
  }

  if (nameIndex == -1) {
    vt.textContent = 'ERROR: Could not find a column called Name. Make sure you copied the header row also.';
    return false;
  }
  if (usernameIndex == -1) {
    vt.textContent = 'ERROR: Could not find a column called Username. Make sure you copied the header row also.';
    return false;
  }

  for (let i = 0; i < words.length; i++) {
    if (!(nameIndex in words[i]) || words[i][nameIndex] === '') {
      vt.textContent = 'ERROR: Could not find the name for the row: ' +
        words[i].join(', ');
      return false;
    }
    if (!(usernameIndex in words[i]) || words[i][usernameIndex] === '') {
      vt.textContent = 'ERROR: Could not find the username for the row: ' +
        words[i].join(', ');
      return false;
    }
    if (words[i].length > headers.length) {
      vt.textContent = 'ERROR: There are some extra columns or text in this row: ' +
        words[i].join(', ');
    }
  }

  vt.textContent = "Found details for " + words.length + " players. Looks good.";
  return true;
}

/**
 * generates an object that matches the proto QRMappingSet with these parsed TSV objects
 * @param {Array<string>} headers - the column headers (first row)
 * @param {Array<Array<string>>} words - the entries in the rest of the table. 
 * @returns {Obejct} of type QRMappingSet from gamedata.proto
 */
function generateConsolidatedObject(headers, words) {

  let nameIndex = -1;
  let usernameIndex = -1;
  let qrCodeIndex = -1;
  let cardSuiteIndex = -1;
  let cardRankIndex = -1;
  for (let i = 0; i < headers.length; i++) {
    if (headers[i].toLowerCase() === 'name') {
      nameIndex = i;
    } else if (headers[i].toLowerCase() === 'username') {
      usernameIndex = i;
    } else if (headers[i].toLowerCase() === 'qrcode') {
      qrCodeIndex = i;
    } else if (headers[i].toLowerCase() === 'cardsuit') {
      cardSuiteIndex = i;
    } else if (headers[i].toLowerCase() === 'cardrank') {
      cardRankIndex = i;
    }
  }

  if (nameIndex == -1 || usernameIndex == -1 ||
    qrCodeIndex == -1 || cardSuiteIndex == -1 ||
    cardRankIndex == -1) {
    return null;
  }

  let obj = { qr_mappings: [] };
  for (let pl of words) {
    let po = {};
    po["username"] = pl[usernameIndex];
    po["display_name"] = pl[nameIndex];
    po["qrcode"] = pl[qrCodeIndex];
    po["card_suit"] = pl[cardSuiteIndex];
    po["card_rank"] = pl[cardRankIndex];
    obj.qr_mappings.push(po);
  }

  return obj;
}