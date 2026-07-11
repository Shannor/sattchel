import { API } from './api.js';
import { Drawer } from './components/drawer.js';
import { MindMap } from './mindmap/mindmap.js';

class App {
    constructor() {
        this.projectId = new URLSearchParams(window.location.search).get('projectId') || '';
        this.api = new API(this.projectId);
        this.drawer = new Drawer();
        
        // Initialize mindmap component, passing API and drawer for interactions
        this.mindmap = new MindMap('#mindmap-container', this.api, this.drawer);
        
        this.init();
    }

    async init() {
        try {
            const goals = await this.api.fetchGoals();
            if (!goals || goals.length === 0) {
                alert('No goals found for this project.');
                return;
            }
            this.mindmap.render(goals);
        } catch (err) {
            console.error('Failed to initialize Sattchel Visualizer:', err);
            alert('Failed to load goals: ' + err.message);
        }
    }
}

// Start the app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.app = new App();
});
