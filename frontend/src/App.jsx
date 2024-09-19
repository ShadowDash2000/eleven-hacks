import {useState} from 'react';
import logo from './assets/images/logo-universal.png';
import './App.css';
import {RegisterAndConfirmAccount, UpdateBridge} from "../wailsjs/go/main/App";
import "https://js.hcaptcha.com/1/api.js";

function App() {
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

    return (
        <div id="App">
            <img src={logo} id="logo" alt="logo"/>
            <div id="input" className="input-box">
                <span>Bridge</span>
                <input id="name" className="input" onChange={updateBridge} autoComplete="off" name="bridge" type="text"/>
                <div class="h-captcha" data-sitekey="3aad1500-7e79-4051-aac5-6852324dab76"></div>
                <button onClick={register}>Register account</button>
            </div>
        </div>
    )
}

export default App
