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

function tabchange() {
    if (document.getElementById("tab-1").checked) {
        stopCamera();
        document.getElementById('cluecontent').classList.replace('hidden', 'visible');
        document.getElementById('scancontent').classList.replace('visible', 'hidden');
    }
    else {
        document.getElementById('cluecontent').classList.replace('visible', 'hidden');
        document.getElementById('scancontent').classList.replace('hidden', 'visible');
        startCamera();
    }
}

function stopCamera() {
    continueTicking = false;
    if (video && video.srcObject && video.srcObject.getTracks() && video.srcObject.getTracks().length > 0) {
        if (video.srcObject.getTracks()[0].readyState = "live") {
            video.srcObject.getTracks()[0].stop();
        }
    }
    const rs = document.getElementById("resumeScanning");
    if (rs.hidden) { rs.hidden = false; }
}

function startCamera() {
    continueTicking = true;
    // Use facingMode: user to attempt to get the front camera on phones
    navigator.mediaDevices.getUserMedia({
        video: {
            facingMode: "environment",
            width: 640,
            height: 640
        }
    }).then(function (stream) {
        video.srcObject = stream;
        video.setAttribute("playsinline", true); // required to tell iOS safari we don't want fullscreen
        video.play();
        requestAnimationFrame(tick);
    });
    const rs = document.getElementById("resumeScanning");
    if (!rs.hidden) { rs.hidden = true; }
}

function resumeScanning() {
    stopCamera();
    startCamera();
}

function flash_scan_success() {
    const sc = document.getElementById("scansuccess");
    sc.className = "msgvisible";
    sc.innerHTML = "Scanned!"
    window.setTimeout(function () { sc.className = "msghidden"; }, 2000);
}

function flash_scan_foreign() {
    const sc = document.getElementById("scansuccess");
    sc.className = "msgvisible";
    sc.innerHTML = "Foreign QR Code!"
    window.setTimeout(function () { sc.className = "msghidden"; }, 2000);
}

function flash_action_msg(str) {
    const sc = document.getElementById("actionlog");
    sc.className = "msgvisible";
    sc.innerHTML = str
    window.setTimeout(function () { sc.className = "msghidden"; }, 2000);
}

function set_life(gs) {
    for (let i = 1; i <= 5; i++) {
        if (gs.life >= i) {
            document.getElementById("heart" + i).classList.replace("heart-dead", "heart-active");
        } else {
            document.getElementById("heart" + i).classList.replace("heart-active", "heart-dead");
        }
    }

    if (gs.has_al) {
        document.getElementById("metalal").classList.replace("hidden", "visible");
    } else {
        document.getElementById("metalal").classList.replace("visible", "hidden");
    }
    if (gs.has_cu) {
        document.getElementById("metalcu").classList.replace("hidden", "visible");
    } else {
        document.getElementById("metalcu").classList.replace("visible", "hidden");
    }
    if (gs.has_sn) {
        document.getElementById("metalsn").classList.replace("hidden", "visible");
    } else {
        document.getElementById("metalsn").classList.replace("visible", "hidden");
    }
    if (gs.has_zn) {
        document.getElementById("metalzn").classList.replace("hidden", "visible");
    } else {
        document.getElementById("metalzn").classList.replace("visible", "hidden");
    }
}

function processResponse(data) {
    console.log(data);
    if (data.hasOwnProperty("GameArtifacts") && data.GameArtifacts.hasOwnProperty("redirectUrl")) {
        window.location.assign(data.GameArtifacts.redirectUrl)
        return;
    }
    if (data.hasOwnProperty("PortHTML")) {
        document.getElementById("cluecontent").innerHTML = data["PortHTML"];
    } else {
        document.getElementById("errormsg").innerHTML = "did not get any clue content as a result";
    }
    if (data.hasOwnProperty("GameArtifacts")) {
        if (data["GameArtifacts"].hasOwnProperty("action")) {
            const msf = data["GameArtifacts"]["action"];
            flash_action_msg(msf);
            // On a correct clue, or on game over, we switch to the first tab
            if (msf == "Correct!" || msf == "Dead!" || msf == "Grabbed Metal!") {
                window.setTimeout(function () {
                    document.getElementById("tab-1").checked = "checked";
                    // programmatically triggering the input does not trigger the event,
                    // so we trigger the handler ourselves.
                    tabchange();
                }, 500);
            } else { // otherwise, keep scanning
                continueTicking = true;
            }
        }
    }
    if (data.hasOwnProperty("State")) {
        set_life(data.State)
    }
}

async function cluescansim_click() {
    if (scannedValidCode.length <= 0) {
        return;
    }
    const scanned = scannedValidCode;
    if (scanned.length > 0) {
        flash_scan_success();
        const postData = new URLSearchParams({ "answer": scanned });
        fetch(PostEndpoint, { method: 'post', body: postData })
            .then(response => {
                if (!response.ok) {
                    response.text().then(p => { document.getElementById('errormsg').textContent = 'Error: ' + p });
                } else {
                    response.json().then(p => processResponse(p));
                }
            }).catch((error) => {
                document.getElementById('errormsg').textContent = 'Error: ' + error;
            });
    }
}

var video = document.createElement("video");
var canvasElement = document.getElementById("canvas");
var canvas = canvasElement.getContext("2d");
var loadingMessage = document.getElementById("loadingMessage");
var outputContainer = document.getElementById("output");
var outputMessage = document.getElementById("outputMessage");
var outputData = document.getElementById("outputData");
var continueTicking = false
var scannedValidCode = "";

function drawLine(begin, end, color) {
    canvas.beginPath();
    canvas.moveTo(begin.x, begin.y);
    canvas.lineTo(end.x, end.y);
    canvas.lineWidth = 4;
    canvas.strokeStyle = color;
    canvas.stroke();
}

function tick() {
    loadingMessage.innerText = "⌛ Loading video..."
    if (!continueTicking) {
        return
    }
    if (video.readyState === video.HAVE_ENOUGH_DATA) {
        loadingMessage.hidden = true;
        canvasElement.hidden = false;
        outputContainer.hidden = false;

        canvasElement.height = video.videoHeight;
        canvasElement.width = video.videoWidth;
        canvas.drawImage(video, 0, 0, canvasElement.width, canvasElement.height);
        var imageData = canvas.getImageData(0, 0, canvasElement.width, canvasElement.height);
        var code = jsQR(imageData.data, imageData.width, imageData.height, {
            inversionAttempts: "dontInvert",
        });
        if (code) {
            drawLine(code.location.topLeftCorner, code.location.topRightCorner, "#FF3B58");
            drawLine(code.location.topRightCorner, code.location.bottomRightCorner, "#FF3B58");
            drawLine(code.location.bottomRightCorner, code.location.bottomLeftCorner, "#FF3B58");
            drawLine(code.location.bottomLeftCorner, code.location.topLeftCorner, "#FF3B58");
            outputMessage.hidden = true;
            outputData.parentElement.hidden = false;
            outputData.innerText = code.data;
            if (code.data.startsWith("https://qr.sd3.in/")) {
                scannedValidCode = code.data;
                stopCamera();
                cluescansim_click();
            } else {
                flash_scan_foreign();
            }
        } else {
            outputMessage.hidden = false;
            outputData.parentElement.hidden = true;
        }
    }
    requestAnimationFrame(tick);
}