import {useEffect, useRef, useState} from 'react';
import { elevenlabs } from "../wailsjs/go/models";
import bgVideo from './assets/videos/in-the-end.mp4';
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
    GetDubbingFiles,
} from "../wailsjs/go/main/App";

const hCaptchaSiteKey = import.meta.env.VITE_H_CAPTCHA_SITE_KEY;

function App() {
    const hCaptchaRef = useRef(null);

    const [savePath, setSavePath] = useState("");
    const [torPath, setTorPath] = useState("");

    const [languages, setLanguages] = useState({});

    const [sourceLanguage, setSourceLanguage] = useState("en");
    const [targetLanguage, setTargetLanguage] = useState("ru");

    const [dubbingFiles, setDubbingFiles] = useState({});

    const updateBridge = (e) => {
        UpdateBridge(e.target.value)
    }

    const chooseFiles = async () => {
        let filePaths = await ChooseFiles();
        await setTokens(filePaths);
    }

    const setTokens = async (filePaths) => {
        for (const filePath of filePaths) {
            const res = await hCaptchaRef.current.execute({ async: true });
            if (res.response) {
                await AddDubbingFile(res.response, filePath);
            }

            await hCaptchaRef.current.resetCaptcha({ async: true });
            await new Promise(resolve => setTimeout(resolve, 2000));
        }
    }

    const startDubbing = () => {
        StartDubbing(sourceLanguage, targetLanguage);
    }

    useEffect(() => {
        (async () => {
            setTorPath(await GetTorPath())
        })();

        (async () => {
            setLanguages(await GetLanguages());
        })();

        window.runtime.EventsOn('LOG', (logMessage) => {
            console.log(logMessage);
        });

        window.runtime.EventsOn('DUBBING.UPDATE', async () => {
            setDubbingFiles(await GetDubbingFiles());
        });

        return () => {
            window.runtime.EventsOff('LOG');
            window.runtime.EventsOff('DUBBING.UPDATE');
        }
    }, []);

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
                <h2>3. Choose language</h2>
                <div className="language">
                    <div>
                        <span>Source language</span>
                        <select value={sourceLanguage} onChange={(e) => setSourceLanguage(e.target.value)}>
                            {Object.entries(languages).map(([key, value]) => (<option value={key}>{value}</option>))}
                        </select>
                    </div>
                    <div>
                        <span>Target language</span>
                        <select value={targetLanguage} onChange={(e) => setTargetLanguage(e.target.value)}>
                            {Object.entries(languages).map(([key, value]) => (<option value={key}>{value}</option>))}
                        </select>
                    </div>
                </div>
            </div>
            <div>
                <h2>4. Start dubbing</h2>
                <button onClick={startDubbing}>Start dubbing</button>
            </div>
            <div>
                <h2>Status</h2>

                <div className="textarea-wrapper">
                    <video className="background-video" autoPlay loop muted>
                        <source src={bgVideo} type="video/mp4"/>
                    </video>
                    <textarea
                        value={Object.entries(dubbingFiles).map(([key, value]) => (
                            dubbingFiles[key].name + " - " + dubbingFiles[key].status + " [Attempt " + dubbingFiles[key].attempt + "]\n"
                        ))}
                        readOnly={true}
                    >
                    </textarea>
                </div>
            </div>
        </div>
    )
}

export default App
