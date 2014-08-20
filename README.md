<h1>Qdab</h1>
Qdab is a simple Dots and Boxes AI. It is written in golang and python2.7. The basic idea is straightforward, UCT plus ANN.<br />
Qdab can reach a high level. I compared Qdab with dabble.exe(http://www.mathstat.dal.ca/~jpg/dabble/) and PRsBoxes.exe(http://www.dianneandpaul.net/PRsBoxes/), the result is Qdab won them in most cases.

<h1>Running</h1>
I have compiled Qdab for Windows and Linux users, please download correct version before running. You can find it in directory <b>release</b>.<br/>

<b>Note: Please make sure you have seen "Server is running at 127.0.0.1:12345" after run Qdab and before any other operation.</b>

<h2>Windows</h2>
Uncompress the .zip file, and double click <b>Qdab.bat</b>. 

<h2>Linux</h2>
First, make sure you have already installed python2.7, gtk, and python-simplejson in your computer.<br />
Second, uncompress the .tar.bz2 file and make Qdab.sh executable.<br />
Finally, go to the directory containing Qdab.sh, run <b>./Qdab.sh</b>.

<h1>Screenshot</h1>
<img src="https://camo.githubusercontent.com/3c1d8fab59aed7c0175f8a982280ea5265f7c633/687474703a2f2f692e696d6775722e636f6d2f4b7270346838392e706e67" />

<h1>Compilation</h1>
If you are interested in this program, you can make some changes and compile your version.<br />
For Linux:<br />
1. Install FANN 2.2.0 and golang1.2.2 (must be version 1.2), the setup files are in directory env, and configure your Go environment.<br />
2. Run the <b>./install.sh</b> to compile all modules whith will make directory bin.<br />
3. Simply copy directory <b>AnnModel</b> from my release version to your directory ./bin or train your own ANN models, to get it you need to understand my code.<br />
4. The binary file needed to run Qdab is <b>./bin/server</b> and <b>./src/guiclient/guiclient.py</b>, please note `pwd` must be directory bin when you run the server.

