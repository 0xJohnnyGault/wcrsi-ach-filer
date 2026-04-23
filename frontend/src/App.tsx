import { useState, useEffect } from 'react';
import './App.css';
import logoSvg from './assets/images/logo.svg';
import { GetConfig, SaveConfig, SelectFolder, ProcessFiles, CheckFiler, CBPFiler, DocFiler, GetActivityLog } from "../wailsjs/go/main/App";

interface LogEntry {
  type: string;
  message: string;
}

interface ProcessResult {
  logs: LogEntry[];
  success: boolean;
}

interface ActivityEntry {
  timestamp: string;
  task: string;
  summary: string;
  status: string;
}

const tasks = [
  {
    id: 'ach',
    title: 'ACH Filer',
    color: 'green',
    description: 'Processes an XLS/XLSX spreadsheet from the source folder. Reads the "Account" column in Row 2 to match account numbers against destination subdirectories. Copies all source files into each matched directory\'s Payments folder.',
    requires: 'XLS/XLSX file with "Account" column in Row 2',
  },
  {
    id: 'check',
    title: 'Check Filer',
    color: 'orange',
    description: 'Scans the source folder for PDF files. Extracts the account number from the last 10 characters of each PDF filename. Matches against destination subdirectories and copies each PDF into the corresponding Payments folder.',
    requires: 'PDF files where the last 10 filename characters are the account number',
  },
  {
    id: 'cbp',
    title: 'CBP Filer',
    color: 'purple',
    description: 'Processes multiple XLS and PDF files grouped by date (YYYY_MM_DD in filenames). Extracts the loan number from the last 10 characters of each XLS filename to match against destination subdirectories. Copies each XLS along with its date-matched PDF deposit slip into the matched directory\'s Payments folder.',
    requires: 'XLS/XLSX files named YYYY_MM_DD_XXXXXXXXXX.xlsx (10-digit loan number) and PDF deposit slips named YYYY_MM_DD_Deposit_Slip.pdf',
  },
  {
    id: 'doc',
    title: 'Doc Filer',
    color: 'teal',
    description: 'Scans the source folder for documents of any type. Extracts the account number from the last 10 characters of each filename (must be all digits). Copies each document directly into the matched destination subdirectory (NOT into a Payments subfolder). Reports an error for files without a valid 10-digit account number.',
    requires: 'Files of any type whose last 10 filename characters (before the extension) are all digits (the account number)',
  },
];

function App() {
  const [sourceFolder, setSourceFolder] = useState('');
  const [destFolder, setDestFolder] = useState('');
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [processing, setProcessing] = useState(false);
  const [activity, setActivity] = useState<ActivityEntry[]>([]);

  function loadActivity() {
    GetActivityLog().then((entries: any) => {
      setActivity(entries || []);
    });
  }

  useEffect(() => {
    GetConfig().then((cfg: any) => {
      if (cfg) {
        setSourceFolder(cfg.sourceFolder || '');
        setDestFolder(cfg.destFolder || '');
      }
    });
    loadActivity();
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

  async function runTask(taskId: string) {
    if (!sourceFolder || !destFolder) return;
    setLogs([]);
    setProcessing(true);
    try {
      let result: ProcessResult;
      switch (taskId) {
        case 'ach':
          result = await ProcessFiles(sourceFolder, destFolder);
          break;
        case 'check':
          result = await CheckFiler(sourceFolder, destFolder);
          break;
        case 'cbp':
          result = await CBPFiler(sourceFolder, destFolder);
          break;
        case 'doc':
          result = await DocFiler(sourceFolder, destFolder);
          break;
        default:
          return;
      }
      setLogs(result.logs || []);
      loadActivity();
    } catch (err: any) {
      setLogs([{ type: 'error', message: `Unexpected error: ${err}` }]);
    } finally {
      setProcessing(false);
    }
  }

  const foldersReady = sourceFolder && destFolder;

  return (
    <div id="App">
      <div className="logo-section">
        <img src={logoSvg} alt="Gohno" className="logo" />
        <span className="rev-label">Rev 7</span>
      </div>

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

      <div className="cards-section">
        {tasks.map((task) => (
          <div key={task.id} className={`card card-${task.color}`}>
            <div className="card-header">
              <h3 className="card-title">{task.title}</h3>
            </div>
            <p className="card-description">{task.description}</p>
            <div className="card-requires">
              <span className="requires-label">Requires:</span> {task.requires}
            </div>
            <button
              className={`btn card-btn card-btn-${task.color}`}
              onClick={() => runTask(task.id)}
              disabled={processing || !foldersReady}
            >
              {processing ? 'Processing...' : `Run ${task.title}`}
            </button>
          </div>
        ))}
      </div>

      {logs.length > 0 && (
        <div className="log-section">
          <h3>Results</h3>
          <div className="log-list">
            {logs.map((log, i) => (
              <div key={i} className={`log-entry log-${log.type}`}>
                {log.type === 'success' ? '\u2713' : '\u2717'} {log.message}
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="activity-section">
        <h3>Activity Log</h3>
        {activity.length === 0 ? (
          <p className="activity-empty">No activity yet</p>
        ) : (
          <div className="activity-list">
            {activity.map((entry, i) => (
              <div key={i} className={`activity-entry activity-${entry.status}`}>
                <span className="activity-time">{entry.timestamp}</span>
                <span className="activity-task">{entry.task}</span>
                <span className="activity-summary">{entry.summary}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export default App;
