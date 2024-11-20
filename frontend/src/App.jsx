import {useEffect, useRef, useState} from 'react';
import { main } from "../wailsjs/go/models";
import HCaptcha from '@hcaptcha/react-hcaptcha';
import './App.css';
import {
    SetSavePath,
    ChooseFiles,
    StartDubbing,
    GetLanguages,
    UpdateBridge,
    AddDubbingFile,
    GetTorPath,
    SetTorPath,
} from "../wailsjs/go/main/App";

const hCaptchaSiteKey = "3aad1500-7e79-4051-aac5-6852324dab76";

function App() {
    const [logs, setLog] = useState("");
    const addLog = (text) => {
        setLog((prev) => prev + text + "\n");
        textAreaRef.current.scrollTop = textAreaRef.current.scrollHeight;
    }

    const textAreaRef = useRef(null);
    const hCaptchaRef = useRef(null);

    const [savePath, setSavePath] = useState("");
    const [torPath, setTorPath] = useState("");
    (async () => {
        setTorPath(await GetTorPath())
    })();

    const updateBridge = (e) => {
        UpdateBridge(e.target.value)
    }

    const chooseFiles = async () => {
        let filePaths = await ChooseFiles();
        addLog("Chosen files: " + filePaths.join(", "))
        await setTokens(filePaths);
    }

    const setTokens = async (filePaths) => {
        for (const filePath of filePaths) {
            const res = await hCaptchaRef.current.execute({ async: true });
            if (res.response) {
                let token = new main.Token();
                token.FilePath = filePath;
                token.Token = res.response;

                await AddDubbingFile(token);

                addLog("Captcha responded with token for the file " + filePath)
            } else {
                addLog("Captcha responded with error for the file " + filePath)
            }

            await hCaptchaRef.current.resetCaptcha({ async: true });
            await new Promise(resolve => setTimeout(resolve, 2000));
        }
    }

    const startDubbing = () => {
        StartDubbing("eng", "ru");
    }

    useEffect(() => {
        window.runtime.EventsOn('LOG', (logMessage) => {
            console.log(logMessage);
            addLog(logMessage);
        });
        return () => window.runtime.EventsOff('LOG')
    });

    const getLanguages = async () => {
        let lang = await GetLanguages();
    }

    return (
        <div id="app">
            <div className="input-box">
                <span>Bridge (use it if Tor is blocked in your country)</span>
                <input className="input" onChange={updateBridge} autoComplete="off" name="bridge"
                       type="text"/>
            </div>
            <div className="input-box">
                <span>Tor path</span>
                <input readOnly={true} value={torPath}/>
                <button onClick={async () => setTorPath(await SetTorPath())}>Select Tor browser folder</button>
            </div>
            <div id="input" className="input-box">
                <h2>1. Select save folder</h2>
                <input readOnly={true} value={savePath}/>
                <button onClick={async () => {
                    setSavePath(await SetSavePath())
                }}>Select save folder
                </button>
            </div>
            <div>
                <h2>2. Select files for dubbing</h2>
                <p>There is no need for manual captcha check. Solve the captcha only if "puzzle" appears.
                    Manual captcha should appear if you're dubbing many videos in a row.</p>
                <button onClick={chooseFiles}>Select files</button>
                <HCaptcha
                    sitekey={hCaptchaSiteKey}
                    ref={hCaptchaRef}
                />
            </div>
            <div>
                <h2>3. Start dubbing</h2>
                <p>If some videos have failed while dubbing, you can start dubbing again. API tokens keep alive while the program is open.</p>
                <button onClick={startDubbing}>Start dubbing</button>
            </div>
            <div>
                <h2>Logs</h2>

                <div className="textarea-wrapper">
                    <video className="background-video" autoPlay loop muted>
                        <source src="src/assets/videos/in-the-end.mp4" type="video/mp4"/>
                    </video>
                    <textarea
                        value={logs}
                        readOnly={true}
                        ref={textAreaRef}
                    >
                    </textarea>
                </div>
            </div>
        </div>
    )
}

export default App
