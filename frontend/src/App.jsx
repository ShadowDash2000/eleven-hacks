import {useEffect, useRef, useState} from 'react';
import bgVideo from './assets/videos/in-the-end.mp4';
import HCaptcha from '@hcaptcha/react-hcaptcha';
import {
    EventsOn,
    EventsOff
} from '../wailsjs/runtime/runtime.js'
import './App.css';
import {
    SetSavePath,
    GetSavePath,
    ChooseFiles,
    StartDubbing,
    GetLanguages,
    UpdateBridge,
    AddDubbingFile,
    GetTorPath,
    SetTorPath,
    GetDubbingFiles,
    SplitVideo,
} from "../wailsjs/go/main/App";

const hCaptchaSiteKey = import.meta.env.VITE_H_CAPTCHA_SITE_KEY;

const
    EventError = "ERROR",
    EventInfo = "INFO",
    EventDubbingUpdate = "DUBBING.UPDATE"

function App() {
    const hCaptchaRef = useRef(null);

    const [savePath, setSavePath] = useState("");
    const [torPath, setTorPath] = useState("");
    const [autoRepeat, setAutoRepeat] = useState(true);

    const [languages, setLanguages] = useState({});

    const [sourceLanguage, setSourceLanguage] = useState("en");
    const [targetLanguage, setTargetLanguage] = useState("ru");

    const [dubbingFiles, setDubbingFiles] = useState({});

    const chooseFiles = async () => {
        let filePaths = await ChooseFiles();
        await setTokens(filePaths);
    }

    const setTokens = async (filePaths) => {
        for (const filePath of filePaths) {
            let ok = false;
            while (!ok) {
                ok = await addDubbingFile(filePath);

                await hCaptchaRef.current.resetCaptcha({async: true});
                await new Promise(resolve => setTimeout(resolve, 2000));

                if (!autoRepeat) break;
            }
        }
    }

    const addDubbingFile = async filePath => {
        const res = await hCaptchaRef.current.execute({ async: true });
        if (res.response) {
            try {
                await AddDubbingFile(res.response, filePath);
                return true;
            } catch (e) {
                return false;
            }
        }

        return false;
    }

    useEffect(() => {
        (async () => {
            setTorPath(await GetTorPath())
        })();

        (async () => {
            setSavePath(await GetSavePath())
        })();

        (async () => {
            setLanguages(await GetLanguages());
        })();

        EventsOn(EventError, (logMessage) => {
            console.log(logMessage);
        });

        EventsOn(EventInfo, (logMessage) => {
            console.log(logMessage);
        });

        EventsOn(EventDubbingUpdate, async () => {
            setDubbingFiles(await GetDubbingFiles());
        });

        return () => {
            EventsOff(EventError);
            EventsOff(EventInfo);
            EventsOff(EventDubbingUpdate);
        }
    }, []);

    return (
        <div id="app">
            <div className="input-box">
                <span>Bridge (use it if Tor is blocked in your country)</span>
                <input className="input" onChange={async (e) => {
                    await UpdateBridge(e.target.value);
                }} autoComplete="off" name="bridge" type="text"/>
            </div>
            <div className="input-box">
                <span>Tor path</span>
                <input readOnly={true} value={torPath}/>
                <button onClick={async () => setTorPath(await SetTorPath())}>Select Tor browser folder</button>
            </div>
            <div id="input" className="input-box">
                <span>Split video(-s)</span>
                <button onClick={async () => {
                    await SplitVideo(220)
                }}>Split video(-s)
                </button>
            </div>
            <div id="input" className="input-box">
                <h2>1. Select save folder</h2>
                <input readOnly={true} value={savePath}/>
                <button onClick={async () => {
                    setSavePath(await SetSavePath());
                }}>Select save folder
                </button>
            </div>
            <div>
                <h2>2. Select files for dubbing</h2>
                <p>There is no need for manual captcha check. Solve the captcha only if "puzzle" appears.
                    Manual captcha should appear if you're dubbing many videos in a row.</p>
                <div className="input-box">
                    <div>
                        <label htmlFor="auto-repeat">Auto-repeat:</label>
                        <input
                            id="auto-repeat"
                            type="checkbox"
                            checked={autoRepeat}
                            onChange={(e) => setAutoRepeat(e.target.checked)}
                        />
                    </div>
                    <button onClick={chooseFiles}>Select files</button>
                    <HCaptcha
                        sitekey={hCaptchaSiteKey}
                        ref={hCaptchaRef}
                    />
                </div>
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
                <button onClick={async () => {
                    await StartDubbing(sourceLanguage, targetLanguage);
                }}>Start dubbing</button>
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
