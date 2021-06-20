const howToRestart = ' In order to restart the application, close the browser tab and run the application (double click the exe file) again.';
let Messages = {
    msg: {
        ApplicationStatesModel_001: 'Trying to restart the application...',
        ApplicationStatesModel_002: 'The application has restarted. This page will be reloaded.',
        ApplicationStatesModel_003: 'Another browser tab should have opened. This tab is terminated.',
        HeaderModel_001: 'Are you sure you want to terminate the application?' + ' If you terminate the application, the drone lands (if it has already taken off) and the application entirely stops.',
        HeaderView_001: 'The application has been terminated.' + howToRestart,
        MainControlModel_001: 'Are you sure you want to stop the application?' + ' If you terminate the application, the video streaming stops and drone lands (if it has already taken off).',
        ModalModel_001: 'To start using the application, please click the \'continue\' button below.',
        SetupModel_001: 'Are you sure you want to update the existing access token?',
        SetupModel_002: 'Are you sure you want to delete the existing access token?',
    },
    err: {
        Common_001: 'The application failed to complete the process. Please check whether the application is running.',
        ApplicationStatesModel_001: 'The application is unavailable.' + 'If it has already stopped, you need to restart it.' + howToRestart,
        MainControlModel_001: 'Can not generate a start key. The signaling server failed to authorize this application or is unavailable.',
        MainControlModel_002: 'Can not start signaling. The signaling server failed to validate the input start key or is unavailable.',
        ModalModel_001: 'The application failed to start. Please check it is running without errors.',
        SetupModel_001: 'The application failed to update the existing access token. The input access token may be invalid.',
    }
};

if (/^ja\b/.test(navigator.language)) {

    const howToRestart_ja = ' アプリケーションを再起動するには、ブラウザタブを閉じて、アプリケーションを再度実行します（exeファイルをダブルクリックします）。';
    Messages = {
        msg: {
            ApplicationStatesModel_001: 'アプリケーションを再起動しようとしています...',
            ApplicationStatesModel_002: 'アプリケーションが再起動しました。このページは再読み込みされます。',
            ApplicationStatesModel_003: '別のブラウザタブが開いているはずです。このタブは終了します。',
            HeaderModel_001: 'アプリケーションを終了してもよろしいですか？' + ' アプリケーションを終了すると、ドローンが着陸し（すでに離陸している場合）、アプリケーションは完全に停止します。',
            HeaderView_001: 'アプリケーションは終了しました。' + howToRestart_ja,
            MainControlModel_001: 'アプリケーションを停止してもよろしいですか？' + ' アプリケーションを終了すると、ビデオストリーミングが停止し、ドローンが着陸します（すでに離陸している場合）。',
            ModalModel_001: 'アプリケーションの使用を開始するには、下の[continue]ボタンをクリックしてください。',
            SetupModel_001: '既存のアクセストークンを更新してもよろしいですか？',
            SetupModel_002: '既存のアクセストークンを削除してもよろしいですか？',
        },
        err: {
            Common_001: 'アプリケーションはプロセスを完了できませんでした。アプリケーションが実行されているかどうかを確認してください。',
            ApplicationStatesModel_001: 'アプリケーションは利用できません。' + 'すでに停止している場合は、再起動する必要があります。' + howToRestart_ja,
            MainControlModel_001: 'スタートキーを生成できません。シグナリングサーバーがこのアプリケーションの承認に失敗したか、使用できません。',
            MainControlModel_002: 'シグナリングを開始できません。シグナリングサーバーがスタートキーの検証に失敗したか、使用できません。',
            ModalModel_001: 'アプリケーションを起動できませんでした。エラーなしで実行されていることを確認してください。',
            SetupModel_001: 'アプリケーションは既存のアクセストークンの更新に失敗しました。入力されたアクセストークンが無効である可能性があります。',
        }
    };
}


export default Messages;