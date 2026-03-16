import { useState, useEffect } from 'react';
import './App.css';
import { GetConfig, SaveConfig, SelectFolder, ProcessFiles, CheckFiler } from "../wailsjs/go/main/App";

interface LogEntry {
  type: string;
  message: string;
}

interface ProcessResult {
  logs: LogEntry[];
  success: boolean;
}

function App() {
  const [sourceFolder, setSourceFolder] = useState('');
  const [destFolder, setDestFolder] = useState('');
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [processing, setProcessing] = useState(false);

  useEffect(() => {
    GetConfig().then((cfg: any) => {
      if (cfg) {
        setSourceFolder(cfg.sourceFolder || '');
        setDestFolder(cfg.destFolder || '');
      }
    });
  }, []);

  async function selectSource() {
    const folder = await SelectFolder("Select Source Folder");
    if (folder) {
      setSourceFolder(folder);
      await SaveConfig({ sourceFolder: folder, destFolder });
    }
  }

  async function selectDest() {
    const folder = await SelectFolder("Select Destination Folder");
    if (folder) {
      setDestFolder(folder);
      await SaveConfig({ sourceFolder, destFolder: folder });
    }
  }

  async function handleProcess() {
    if (!sourceFolder || !destFolder) return;
    setLogs([]);
    setProcessing(true);
    try {
      const result: ProcessResult = await ProcessFiles(sourceFolder, destFolder);
      setLogs(result.logs || []);
    } catch (err: any) {
      setLogs([{ type: 'error', message: `Unexpected error: ${err}` }]);
    } finally {
      setProcessing(false);
    }
  }

  async function handleCheckFiler() {
    if (!sourceFolder || !destFolder) return;
    setLogs([]);
    setProcessing(true);
    try {
      const result: ProcessResult = await CheckFiler(sourceFolder, destFolder);
      setLogs(result.logs || []);
    } catch (err: any) {
      setLogs([{ type: 'error', message: `Unexpected error: ${err}` }]);
    } finally {
      setProcessing(false);
    }
  }

  return (
    <div id="App">
      <h1>Gohno</h1>

      <div className="folder-section">
        <div className="folder-row">
          <button className="btn" onClick={selectSource}>Select Source Folder</button>
          <span className="folder-path">{sourceFolder || 'No folder selected'}</span>
        </div>
        <div className="folder-row">
          <button className="btn" onClick={selectDest}>Select Destination Folder</button>
          <span className="folder-path">{destFolder || 'No folder selected'}</span>
        </div>
      </div>

      <div className="action-section">
        <button
          className="btn btn-process"
          onClick={handleProcess}
          disabled={processing || !sourceFolder || !destFolder}
        >
          {processing ? 'Processing...' : 'ACH Filer'}
        </button>
        <button
          className="btn btn-check"
          onClick={handleCheckFiler}
          disabled={processing || !sourceFolder || !destFolder}
        >
          {processing ? 'Processing...' : 'Check Filer'}
        </button>
      </div>

      {logs.length > 0 && (
        <div className="log-section">
          <h3>Results</h3>
          <div className="log-list">
            {logs.map((log, i) => (
              <div key={i} className={`log-entry log-${log.type}`}>
                {log.type === 'success' ? '✓' : '✗'} {log.message}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
