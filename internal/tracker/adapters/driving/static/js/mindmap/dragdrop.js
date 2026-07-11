export class DragDrop {
    constructor(mindmap) {
        this.mindmap = mindmap;
    }

    handleDragStart(e, id) {
        const node = this.mindmap.goalsMap[id];
        if (!node || !node.parent || !node.parent.targetId) {
            // It's the root goal!
            e.preventDefault();
            alert('The root goal cannot be moved.');
            return;
        }
        e.dataTransfer.setData('text/plain', id);
        e.target.classList.add('dragging');
    }

    handleDragOver(e) {
        e.preventDefault();
        const card = e.target.closest('.goal-card');
        if (card && !card.classList.contains('dragging')) {
            card.classList.add('drag-hover');
        }
    }

    handleDragLeave(e) {
        const card = e.target.closest('.goal-card');
        if (card) {
            card.classList.remove('drag-hover');
        }
    }

    handleDragEnd(e) {
        e.target.classList.remove('dragging');
        document.querySelectorAll('.goal-card').forEach(c => c.classList.remove('drag-hover'));
    }

    async handleDrop(e, targetId) {
        e.preventDefault();
        const card = e.target.closest('.goal-card');
        if (card) {
            card.classList.remove('drag-hover');
        }

        const childId = e.dataTransfer.getData('text/plain');
        if (!childId || childId === targetId) return;

        try {
            await this.mindmap.api.moveGoal(childId, targetId);
            // Refresh mindmap nodes and connections
            const goals = await this.mindmap.api.fetchGoals();
            this.mindmap.render(goals);
        } catch (err) {
            alert('Failed to move goal: ' + err.message);
            document.querySelectorAll('.goal-card').forEach(c => {
                c.classList.remove('dragging');
                c.classList.remove('drag-hover');
            });
        }
    }
}
