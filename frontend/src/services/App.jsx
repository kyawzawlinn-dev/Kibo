// import React, { useState, useEffect, useCallback, useRef } from 'react';
// import { getAuth, signInAnonymously, signInWithCustomToken, onAuthStateChanged } from 'firebase/auth';
// import { initializeApp } from 'firebase/app';
// import { getFirestore, doc, onSnapshot, collection, query, setDoc, addDoc, serverTimestamp } from 'firebase/firestore';
// import { Line } from 'react-chartjs-2';
// import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend } from 'chart.js';
// import { Send, User, Bot, Loader, Heart, MessageSquare } from 'lucide-react'; // Using lucide-react for icons

// // Initialize ChartJS components
// ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend);

// // --- FIREBASE AND API CONFIGURATION ---
// const firebaseConfig = typeof __firebase_config !== 'undefined' ? JSON.parse(__firebase_config) : null;
// const appId = typeof __app_id !== 'undefined' ? __app_id : 'default-app-id';
// const initialAuthToken = typeof __initial_auth_token !== 'undefined' ? __initial_auth_token : null;
// const API_KEY = ""; // Placeholder for Gemini API key

// let db;
// let auth;
// let app;

// // Utility functions for Firebase initialization (outside of React component scope)
// const initializeFirebase = () => {
//   if (firebaseConfig && !app) {
//     app = initializeApp(firebaseConfig);
//     db = getFirestore(app);
//     auth = getAuth(app);
//     // setLogLevel('Debug'); // Uncomment for debugging
//     return { db, auth };
//   }
//   return { db, auth };
// };

// // --- CHAT PAGE COMPONENTS ---

// /**
//  * Handles the API call to the Gemini model with Google Search grounding.
//  */
// const generateGeminiContent = async (history) => {
//     const userQuery = history[history.length - 1].text;
//     const systemInstruction = {
//         parts: [{ text: "You are a friendly and informative assistant focused on health, wellness, and general knowledge. You must use Google Search to find up-to-date, grounded information whenever answering factual queries." }]
//     };

//     const payload = {
//         contents: [{ parts: [{ text: userQuery }] }],
//         tools: [{ "google_search": {} }],
//         systemInstruction: systemInstruction,
//     };

//     const apiUrl = `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-preview-09-2025:generateContent?key=${API_KEY}`;
    
//     // Exponential backoff retry logic
//     for (let attempt = 0; attempt < 5; attempt++) {
//         try {
//             const response = await fetch(apiUrl, {
//                 method: 'POST',
//                 headers: { 'Content-Type': 'application/json' },
//                 body: JSON.stringify(payload)
//             });

//             if (!response.ok) {
//                 // If 429 (Too Many Requests), proceed to retry
//                 if (response.status === 429 && attempt < 4) {
//                     const delay = Math.pow(2, attempt) * 1000 + Math.random() * 1000;
//                     await new Promise(resolve => setTimeout(resolve, delay));
//                     continue; // Retry the request
//                 }
//                 throw new Error(`API call failed with status: ${response.status}`);
//             }

//             const result = await response.json();
//             const candidate = result.candidates?.[0];

//             if (candidate && candidate.content?.parts?.[0]?.text) {
//                 const text = candidate.content.parts[0].text;
//                 let sources = [];
//                 const groundingMetadata = candidate.groundingMetadata;

//                 if (groundingMetadata && groundingMetadata.groundingAttributions) {
//                     sources = groundingMetadata.groundingAttributions
//                         .map(attribution => ({
//                             uri: attribution.web?.uri,
//                             title: attribution.web?.title,
//                         }))
//                         .filter(source => source.uri && source.title);
//                 }
//                 return { text, sources };

//             } else {
//                 throw new Error("Invalid or empty response from Gemini API.");
//             }
//         } catch (error) {
//             console.error("Gemini API Error:", error);
//             if (attempt === 4) return { text: "I'm sorry, I was unable to connect to the assistant. Please try again later.", sources: [] };
//         }
//     }
// };


// /**
//  * Renders a single message bubble in the chat.
//  */
// const ChatMessage = ({ message }) => {
//     const isUser = message.role === 'user';
//     const bgColor = isUser ? 'bg-indigo-500/10' : 'bg-gray-100 dark:bg-gray-800';
//     const textColor = isUser ? 'text-indigo-800 dark:text-indigo-200' : 'text-gray-900 dark:text-gray-50';
//     const alignment = isUser ? 'self-end' : 'self-start';
//     const Icon = isUser ? User : Bot;

//     return (
//         <div className={`flex w-full ${isUser ? 'justify-end' : 'justify-start'} mb-4`}>
//             <div className={`flex items-start max-w-3/4 p-4 rounded-xl shadow-md ${bgColor} ${alignment}`}>
//                 <div className={`p-2 rounded-full mr-3 ${isUser ? 'bg-indigo-500 text-white' : 'bg-white dark:bg-gray-700 text-gray-700 dark:text-white'}`}>
//                     <Icon size={18} />
//                 </div>
//                 <div>
//                     <p className={`text-sm font-semibold mb-1 ${isUser ? 'text-indigo-600 dark:text-indigo-400' : 'text-gray-600 dark:text-gray-300'}`}>
//                         {isUser ? 'You' : 'AI Assistant'}
//                     </p>
//                     <div className={`whitespace-pre-wrap ${textColor}`}>
//                         {message.text}
//                     </div>
//                     {message.sources && message.sources.length > 0 && (
//                         <div className="mt-3 pt-3 border-t border-gray-300 dark:border-gray-700 text-xs text-gray-500 dark:text-gray-400">
//                             <p className="font-medium mb-1">Sources:</p>
//                             <ul className="list-disc list-inside space-y-1">
//                                 {message.sources.map((source, index) => (
//                                     <li key={index}>
//                                         <a 
//                                             href={source.uri} 
//                                             target="_blank" 
//                                             rel="noopener noreferrer" 
//                                             className="text-indigo-600 hover:text-indigo-400 dark:text-indigo-400 dark:hover:text-indigo-200 truncate block"
//                                             title={source.title}
//                                         >
//                                             {source.title}
//                                         </a>
//                                     </li>
//                                 ))}
//                             </ul>
//                         </div>
//                     )}
//                 </div>
//             </div>
//         </div>
//     );
// };

// /**
//  * Dedicated component for the AI Chat feature.
//  */
// const ChatPage = ({ db, userId }) => {
//     const [chatHistory, setChatHistory] = useState([]);
//     const [input, setInput] = useState('');
//     const [isLoading, setIsLoading] = useState(false);
//     const chatContainerRef = useRef(null);

//     // Scroll to bottom on new message
//     useEffect(() => {
//         if (chatContainerRef.current) {
//             chatContainerRef.current.scrollTop = chatContainerRef.current.scrollHeight;
//         }
//     }, [chatHistory]);

//     // Load history from Firestore
//     useEffect(() => {
//         if (!db || !userId) return;

//         const path = `artifacts/${appId}/users/${userId}/chats`;
//         const q = query(collection(db, path));
        
//         const unsubscribe = onSnapshot(q, (snapshot) => {
//             const history = snapshot.docs
//                 .map(doc => doc.data())
//                 .sort((a, b) => a.timestamp.toDate() - b.timestamp.toDate()) // Sort by timestamp
//                 .map(item => ({
//                     role: item.role,
//                     text: item.text,
//                     sources: item.sources || [],
//                     id: item.id
//                 }));
//             setChatHistory(history);
//         }, (error) => {
//             console.error("Error fetching chat history:", error);
//         });

//         return () => unsubscribe();
//     }, [db, userId]);

//     // Save message to Firestore
//     const saveMessage = async (message) => {
//         if (!db || !userId) return;
//         try {
//             const path = `artifacts/${appId}/users/${userId}/chats`;
//             await addDoc(collection(db, path), {
//                 ...message,
//                 timestamp: serverTimestamp(),
//             });
//         } catch (error) {
//             console.error("Error saving message to Firestore:", error);
//         }
//     };


//     const handleSend = async (e) => {
//         e.preventDefault();
//         if (!input.trim() || isLoading) return;

//         const newUserMessage = { role: 'user', text: input.trim(), id: crypto.randomUUID() };
        
//         // Optimistically update UI and save user message
//         setChatHistory(prev => [...prev, newUserMessage]);
//         await saveMessage(newUserMessage);

//         setInput('');
//         setIsLoading(true);

//         // Fetch response from Gemini
//         const responseHistory = [...chatHistory, newUserMessage];
//         const { text, sources } = await generateGeminiContent(responseHistory);

//         const newBotMessage = { role: 'model', text: text, sources: sources, id: crypto.randomUUID() };
        
//         // Update UI with bot message and save it
//         setChatHistory(prev => [...prev.filter(m => m.id !== newUserMessage.id), newUserMessage, newBotMessage]); // Ensure user message is correct before bot message
//         await saveMessage(newBotMessage);

//         setIsLoading(false);
//     };

//     return (
//         <div className="flex flex-col h-full bg-white dark:bg-gray-900 rounded-xl shadow-lg p-6">
//             <h2 className="text-3xl font-bold text-gray-900 dark:text-white mb-6 border-b pb-3 border-gray-200 dark:border-gray-700">
//                 AI Health Assistant Chat
//             </h2>
            
//             <div ref={chatContainerRef} className="flex-grow overflow-y-auto space-y-4 mb-4 pr-2 custom-scrollbar">
//                 {chatHistory.length === 0 && !isLoading ? (
//                     <div className="text-center text-gray-500 dark:text-gray-400 mt-20">
//                         <MessageSquare className="w-12 h-12 mx-auto mb-3" />
//                         <p>Start a conversation! Ask about your health data, nutrition, or wellness tips.</p>
//                     </div>
//                 ) : (
//                     chatHistory.map((msg, index) => (
//                         <ChatMessage key={index} message={msg} />
//                     ))
//                 )}
//                 {isLoading && (
//                     <div className="flex justify-start mb-4">
//                         <div className="flex items-center p-3 rounded-xl bg-gray-100 dark:bg-gray-800 shadow-md">
//                             <Loader className="w-5 h-5 animate-spin mr-2 text-indigo-500" />
//                             <span className="text-sm text-gray-700 dark:text-gray-300">Assistant is typing...</span>
//                         </div>
//                     </div>
//                 )}
//             </div>

//             <form onSubmit={handleSend} className="flex p-4 bg-gray-50 dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 rounded-b-xl">
//                 <input
//                     type="text"
//                     value={input}
//                     onChange={(e) => setInput(e.target.value)}
//                     placeholder="Ask the AI a health question..."
//                     className="flex-grow p-3 border border-gray-300 dark:border-gray-600 rounded-l-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:bg-gray-900 dark:text-white"
//                     disabled={isLoading}
//                 />
//                 <button
//                     type="submit"
//                     className="bg-indigo-600 hover:bg-indigo-700 text-white p-3 rounded-r-lg shadow-md transition duration-150 ease-in-out disabled:bg-indigo-400 flex items-center justify-center"
//                     disabled={isLoading || !input.trim()}
//                 >
//                     <Send size={20} />
//                 </button>
//             </form>
//         </div>
//     );
// };

// // --- BODY RECORD PAGE COMPONENTS ---

// /**
//  * Component to handle form input for new health records.
//  */
// const RecordForm = ({ onAddRecord, userId }) => {
//     const [date, setDate] = useState(new Date().toISOString().slice(0, 10));
//     const [value, setValue] = useState('');
//     const [type, setType] = useState('weight');
//     const [message, setMessage] = useState('');

//     const handleSubmit = (e) => {
//         e.preventDefault();
//         const numericValue = parseFloat(value);

//         if (!userId) {
//             setMessage({ type: 'error', text: 'Authentication is not ready. Please wait.' });
//             return;
//         }

//         if (isNaN(numericValue) || numericValue <= 0) {
//             setMessage({ type: 'error', text: 'Please enter a valid positive number.' });
//             return;
//         }
        
//         onAddRecord({
//             date,
//             value: numericValue,
//             type,
//             timestamp: serverTimestamp()
//         });

//         setValue('');
//         setMessage({ type: 'success', text: 'Record added successfully!' });
//         setTimeout(() => setMessage(''), 3000); // Clear message after 3 seconds
//     };

//     const types = [
//         { key: 'weight', label: 'Weight (kg)', min: 30, max: 200 },
//         { key: 'sleep', label: 'Sleep (hours)', min: 0.5, max: 15 },
//         { key: 'water', label: 'Water (liters)', min: 0.1, max: 10 },
//     ];

//     const currentType = types.find(t => t.key === type);

//     return (
//         <div className="p-6 bg-white dark:bg-gray-800 rounded-xl shadow-lg">
//             <h3 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">Add New Record</h3>
//             <form onSubmit={handleSubmit} className="space-y-4">
//                 <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
//                     <div>
//                         <label htmlFor="recordType" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Metric</label>
//                         <select
//                             id="recordType"
//                             value={type}
//                             onChange={(e) => setType(e.target.value)}
//                             className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-700 shadow-sm p-2 dark:bg-gray-900 dark:text-white"
//                         >
//                             {types.map(t => (
//                                 <option key={t.key} value={t.key}>{t.label}</option>
//                             ))}
//                         </select>
//                     </div>
//                     <div>
//                         <label htmlFor="date" className="block text-sm font-medium text-gray-700 dark:text-gray-300">Date</label>
//                         <input
//                             type="date"
//                             id="date"
//                             value={date}
//                             onChange={(e) => setDate(e.target.value)}
//                             className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-700 shadow-sm p-2 dark:bg-gray-900 dark:text-white"
//                             required
//                             max={new Date().toISOString().slice(0, 10)}
//                         />
//                     </div>
//                 </div>
//                 <div>
//                     <label htmlFor="value" className="block text-sm font-medium text-gray-700 dark:text-gray-300">{currentType.label} Value</label>
//                     <input
//                         type="number"
//                         id="value"
//                         value={value}
//                         onChange={(e) => setValue(e.target.value)}
//                         placeholder={`Enter ${currentType.label}`}
//                         min={currentType.min}
//                         max={currentType.max}
//                         step="0.1"
//                         className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-700 shadow-sm p-2 dark:bg-gray-900 dark:text-white"
//                         required
//                     />
//                 </div>
//                 <button
//                     type="submit"
//                     className="w-full py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition duration-150 ease-in-out"
//                 >
//                     Log Record
//                 </button>
//             </form>
//             {message && (
//                 <div className={`mt-3 p-3 text-center rounded-md ${message.type === 'error' ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'}`}>
//                     {message.text}
//                 </div>
//             )}
//         </div>
//     );
// };

// /**
//  * Component to display health data visualization using a Line chart.
//  */
// const DataVisualization = ({ records }) => {
//     const [selectedType, setSelectedType] = useState('weight');
    
//     if (records.length === 0) {
//         return (
//             <div className="text-center p-10 bg-white dark:bg-gray-800 rounded-xl shadow-lg">
//                 <Heart className="w-12 h-12 mx-auto text-indigo-400 mb-3" />
//                 <p className="text-gray-500 dark:text-gray-400">No records found. Please add a record to see the visualization.</p>
//             </div>
//         );
//     }

//     const filteredData = records
//         .filter(record => record.type === selectedType)
//         .sort((a, b) => new Date(a.date) - new Date(b.date));

//     // Get the last 7 unique days for plotting
//     const uniqueDates = [...new Set(filteredData.map(r => r.date))].slice(-7);

//     const chartData = {
//         labels: uniqueDates,
//         datasets: [
//             {
//                 label: selectedType.charAt(0).toUpperCase() + selectedType.slice(1),
//                 data: uniqueDates.map(date => {
//                     // Use the latest value recorded on that date
//                     const recordsOnDate = filteredData.filter(r => r.date === date);
//                     return recordsOnDate.length > 0 ? recordsOnDate[recordsOnDate.length - 1].value : null;
//                 }),
//                 borderColor: '#4f46e5',
//                 backgroundColor: 'rgba(79, 70, 229, 0.1)',
//                 fill: true,
//                 tension: 0.3,
//                 pointRadius: 6,
//                 pointHoverRadius: 8,
//             },
//         ],
//     };

//     const options = {
//         responsive: true,
//         plugins: {
//             legend: {
//                 position: 'top',
//                 labels: {
//                     color: document.documentElement.classList.contains('dark') ? '#ccc' : '#333',
//                 },
//             },
//             title: {
//                 display: true,
//                 text: `Last 7 Days of ${selectedType.charAt(0).toUpperCase() + selectedType.slice(1)}`,
//                 color: document.documentElement.classList.contains('dark') ? '#fff' : '#1f2937',
//             },
//         },
//         scales: {
//             x: {
//                 title: {
//                     display: true,
//                     text: 'Date',
//                     color: document.documentElement.classList.contains('dark') ? '#ccc' : '#333',
//                 },
//                 ticks: {
//                     color: document.documentElement.classList.contains('dark') ? '#aaa' : '#666',
//                 },
//                 grid: {
//                     color: document.documentElement.classList.contains('dark') ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.1)',
//                 }
//             },
//             y: {
//                 title: {
//                     display: true,
//                     text: 'Value',
//                     color: document.documentElement.classList.contains('dark') ? '#ccc' : '#333',
//                 },
//                 ticks: {
//                     color: document.documentElement.classList.contains('dark') ? '#aaa' : '#666',
//                 },
//                 grid: {
//                     color: document.documentElement.classList.contains('dark') ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.1)',
//                 }
//             },
//         },
//     };

//     return (
//         <div className="p-6 bg-white dark:bg-gray-800 rounded-xl shadow-lg h-full">
//             <div className="flex justify-between items-center mb-4">
//                 <h3 className="text-xl font-semibold text-gray-900 dark:text-white">Progress Chart</h3>
//                 <div>
//                     <label htmlFor="viewMetric" className="text-sm font-medium text-gray-700 dark:text-gray-300 mr-2">View:</label>
//                     <select
//                         id="viewMetric"
//                         value={selectedType}
//                         onChange={(e) => setSelectedType(e.target.value)}
//                         className="rounded-md border-gray-300 dark:border-gray-700 shadow-sm p-1 text-sm dark:bg-gray-900 dark:text-white"
//                     >
//                         <option value="weight">Weight</option>
//                         <option value="sleep">Sleep</option>
//                         <option value="water">Water</option>
//                     </select>
//                 </div>
//             </div>
//             <div className="h-96">
//                 {filteredData.length > 0 ? (
//                     <Line data={chartData} options={options} />
//                 ) : (
//                     <div className="flex items-center justify-center h-full text-gray-500 dark:text-gray-400">
//                         No data to display for {selectedType}.
//                     </div>
//                 )}
//             </div>
//         </div>
//     );
// };

// /**
//  * The container for the Body Record feature.
//  */
// const BodyRecordPage = ({ db, userId }) => {
//     const [records, setRecords] = useState([]);

//     const addRecord = useCallback(async (newRecord) => {
//         if (!db || !userId) {
//             console.error("Database not initialized or user not authenticated.");
//             return;
//         }
//         try {
//             const path = `artifacts/${appId}/users/${userId}/records`;
//             // Add a new document to the 'records' collection
//             await addDoc(collection(db, path), newRecord);
//         } catch (e) {
//             console.error("Error adding document: ", e);
//         }
//     }, [db, userId]);


//     useEffect(() => {
//         if (!db || !userId) return;

//         const path = `artifacts/${appId}/users/${userId}/records`;
//         const q = query(collection(db, path));
        
//         // Listen for real-time updates
//         const unsubscribe = onSnapshot(q, (snapshot) => {
//             const fetchedRecords = snapshot.docs.map(doc => ({
//                 id: doc.id,
//                 ...doc.data(),
//                 // Firestore timestamp objects need to be converted for easy sorting/display
//                 timestamp: doc.data().timestamp ? doc.data().timestamp.toDate() : new Date(),
//             }));
//             setRecords(fetchedRecords);
//         }, (error) => {
//             console.error("Error fetching records:", error);
//         });

//         return () => unsubscribe(); // Cleanup listener on unmount
//     }, [db, userId]);


//     return (
//         <div className="space-y-6">
//             <h2 className="text-3xl font-bold text-gray-900 dark:text-white mb-6 border-b pb-3 border-gray-200 dark:border-gray-700">
//                 Health Data Dashboard
//             </h2>
//             <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
//                 <div className="lg:col-span-1">
//                     <RecordForm onAddRecord={addRecord} userId={userId} />
//                 </div>
//                 <div className="lg:col-span-2">
//                     <DataVisualization records={records} />
//                 </div>
//             </div>
//         </div>
//     );
// };


// /**
//  * Sidebar component for navigation.
//  */
// const Sidebar = ({ currentPage, onNavigate, userId }) => {
//     const navItems = [
//         { id: 'bodyrecord', label: 'BodyRecord', icon: Heart },
//         { id: 'chats', label: 'AI Chat', icon: MessageSquare },
//     ];
    
//     return (
//         <div className="w-64 bg-gray-50 dark:bg-gray-800 p-6 flex flex-col rounded-xl shadow-lg h-full">
//             <div className="mb-8">
//                 <h1 className="text-2xl font-extrabold text-indigo-600 dark:text-indigo-400">Kibo App</h1>
//                 <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Health & Intelligence Platform</p>
//             </div>

//             <nav className="flex-grow space-y-2">
//                 {navItems.map(item => {
//                     const isActive = currentPage === item.id;
//                     return (
//                         <button
//                             key={item.id}
//                             onClick={() => onNavigate(item.id)}
//                             className={`flex items-center w-full p-3 rounded-lg text-left transition duration-150 ease-in-out ${
//                                 isActive 
//                                     ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900 dark:text-indigo-300 font-semibold shadow-inner'
//                                     : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-gray-700'
//                             }`}
//                         >
//                             <item.icon className="w-5 h-5 mr-3" />
//                             {item.label}
//                         </button>
//                     );
//                 })}
//             </nav>

//             <div className="mt-8 pt-4 border-t border-gray-200 dark:border-gray-700">
//                 <p className="text-xs text-gray-500 dark:text-gray-400 mb-2 font-medium">User ID:</p>
//                 <div className="text-xs text-gray-700 dark:text-gray-300 break-all p-2 bg-gray-100 dark:bg-gray-700 rounded-md select-all">
//                     {userId || 'Loading...'}
//                 </div>
//             </div>
//         </div>
//     );
// };


// /**
//  * Main application component.
//  */
// const App = () => {
//     const [currentPage, setCurrentPage] = useState('bodyrecord');
//     const [isAuthReady, setIsAuthReady] = useState(false);
//     const [userId, setUserId] = useState(null);
//     const [dbInstance, setDbInstance] = useState(null);

//     // 1. Initialize Firebase and Authentication
//     useEffect(() => {
//         const { db: initializedDb, auth: initializedAuth } = initializeFirebase();
//         setDbInstance(initializedDb);

//         // Function to handle sign-in
//         const doSignIn = async () => {
//             if (initialAuthToken) {
//                 await signInWithCustomToken(initializedAuth, initialAuthToken)
//                     .catch(e => {
//                         console.error("Custom token sign-in failed:", e);
//                         // Fallback to anonymous if custom token fails
//                         return signInAnonymously(initializedAuth);
//                     });
//             } else {
//                 await signInAnonymously(initializedAuth);
//             }
//         };

//         // Listen for auth state changes
//         const unsubscribeAuth = onAuthStateChanged(initializedAuth, (user) => {
//             if (user) {
//                 setUserId(user.uid);
//             } else {
//                 // Should theoretically not happen after initial sign-in attempt, but set to null just in case.
//                 setUserId(null);
//             }
//             setIsAuthReady(true);
//         });

//         doSignIn();

//         return () => {
//             unsubscribeAuth();
//         };
//     }, []);

//     const renderPage = () => {
//         if (!isAuthReady) {
//             return (
//                 <div className="flex flex-col items-center justify-center h-full">
//                     <Loader className="w-10 h-10 animate-spin text-indigo-500 mb-4" />
//                     <p className="text-lg text-gray-600 dark:text-gray-300">Connecting to database...</p>
//                 </div>
//             );
//         }

//         switch (currentPage) {
//             case 'bodyrecord':
//                 return <BodyRecordPage db={dbInstance} userId={userId} />;
//             case 'chats':
//                 return <ChatPage db={dbInstance} userId={userId} />;
//             default:
//                 return <BodyRecordPage db={dbInstance} userId={userId} />;
//         }
//     };

//     return (
//         <div className="min-h-screen bg-gray-100 dark:bg-gray-900 p-8 font-sans transition-colors duration-300">
//             <style>{`
//                 /* Custom Scrollbar for Chat */
//                 .custom-scrollbar::-webkit-scrollbar {
//                     width: 8px;
//                 }
//                 .custom-scrollbar::-webkit-scrollbar-thumb {
//                     background-color: #d1d5db; /* Light gray for thumb */
//                     border-radius: 4px;
//                 }
//                 .custom-scrollbar::-webkit-scrollbar-track {
//                     background-color: #f3f4f6; /* Lighter background */
//                 }
//                 /* Dark mode adjustments */
//                 .dark .custom-scrollbar::-webkit-scrollbar-thumb {
//                     background-color: #4b5563; /* Darker gray for thumb in dark mode */
//                 }
//                 .dark .custom-scrollbar::-webkit-scrollbar-track {
//                     background-color: #1f2937; /* Darker background */
//                 }
//             `}</style>
//             <div className="max-w-7xl mx-auto flex h-[calc(100vh-4rem)] space-x-8">
//                 <Sidebar currentPage={currentPage} onNavigate={setCurrentPage} userId={userId} />
//                 <main className="flex-grow p-6 bg-white dark:bg-gray-900 rounded-xl shadow-2xl overflow-y-auto">
//                     {renderPage()}
//                 </main>
//             </div>
//         </div>
//     );
// };

// export default App;