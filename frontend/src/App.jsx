import { useState } from 'react';
import logo from './assets/images/logo-universal.png';
import './App.css';
import { RegisterAndConfirmAccount, CreateDubbing, GetLanguages, UpdateBridge } from "../wailsjs/go/main/App";
import "https://js.hcaptcha.com/1/api.js";

function App() {
    const [logs, setLog] = useState("");
    const addLog = (text) => setLog(logs + text + "\n");

    function updateBridge(e) {
        UpdateBridge(e.target.value)
    }

    function register() {
        let iframe = document.querySelector('iframe');
        let captchaResponse = iframe.dataset.hcaptchaResponse;
        if (captchaResponse) {
            RegisterAndConfirmAccount(captchaResponse);
        }
    }

    function dubDub() {
        CreateDubbing()
    }

    window.runtime.EventsOn('LOG', (logMessage) => {
        console.log(logMessage);
        addLog(logMessage);
    });

    async function getLanguages() {
        let lang = await GetLanguages();
    }

    return (
        <div id="app">
            <div className="input-box">
                <span>Bridge (use it if Tor is blocked in your country)</span>
                <input id="name" className="input" onChange={updateBridge} autoComplete="off" name="bridge" type="text" />
            </div>
            <div id="input" className="input-box">
                <h2>1. Create an account</h2>
                <div className="h-captcha" data-sitekey="3aad1500-7e79-4051-aac5-6852324dab76"></div>
                <button onClick={register}>Register an account</button>
            </div>
            <div>
                <h2>2. Dub a video</h2>
                <button onClick={dubDub}>Dub-dub!</button>
            </div>
            <div>
                <h2>Logs</h2>
                <textarea value={logs} readOnly={true} rows={10} cols={50}></textarea>
            </div>
        </div>
    )
}

export default App
